package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"wesmk/internal/model"
	"wesmk/internal/state"
	"wesmk/internal/store"
)

const (
	maxStateBytes           = 2 << 20
	maxStateWritesPerMinute = 120
)

var (
	errStateNotInitialized = errors.New("state not initialized")
	errStateTooLarge       = errors.New("state payload too large")
)

func scopeFor(user *model.User) (access state.Access, scoped bool, linked bool) {
	if !state.IsScopedRole(user.Role) {
		return state.Access{}, false, true
	}
	if user.StudentProfileID == "" {
		return state.Access{}, true, false
	}
	return state.Access{Role: user.Role, StudentID: user.StudentProfileID}, true, true
}

func decodeState(data []byte) (map[string]any, error) {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	return document, nil
}

// updateState applies a server-authoritative mutation inside one locked
// transaction and returns its result. Lost updates and double settlement are
// prevented by the row lock in Postgres.UpdateState.
func (s *Server) updateState(w http.ResponseWriter, r *http.Request, apply func(map[string]any) (any, error)) {
	actorID, role := "", ""
	if auth, ok := requestAuthFromContext(r.Context()); ok {
		actorID, role = auth.user.ID, auth.user.Role
	}
	audit := store.StateAudit{
		ActorID: actorID,
		Role:    role,
		Action:  r.Method + " " + r.URL.Path,
	}
	var result any
	updated, err := s.storeDB.UpdateStateAudited(r.Context(), audit, func(raw json.RawMessage) (json.RawMessage, error) {
		document, err := decodeState(raw)
		if err != nil {
			return nil, err
		}
		value, err := apply(document)
		if err != nil {
			return nil, err
		}
		result = value
		return json.Marshal(document)
	})
	if err != nil {
		writeStateError(w, err)
		return
	}
	s.syncDomainShadow(r.Context(), updated)
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}

// syncDomainShadow keeps the normalized tables current without making them the
// read model yet. A failure is logged but never blocks the request: the JSON
// blob is still authoritative.
func (s *Server) syncDomainShadow(ctx context.Context, updated json.RawMessage) {
	var doc map[string]any
	if err := json.Unmarshal(updated, &doc); err != nil {
		return
	}
	if err := s.storeDB.SyncDomainFromState(ctx, doc); err != nil {
		log.Printf("domain shadow sync failed: %v", err)
	}
}

func writeStateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNoState):
		writeError(w, http.StatusConflict, "Data sekolah belum diinisialisasi di server.")
	case errors.Is(err, state.ErrNotFound):
		writeError(w, http.StatusNotFound, "Data yang diminta tidak ditemukan.")
	case errors.Is(err, state.ErrConflict):
		writeError(w, http.StatusConflict, "Data sudah berubah. Muat ulang lalu coba lagi.")
	case errors.Is(err, state.ErrInvalid):
		writeError(w, http.StatusBadRequest, "Data yang dikirim tidak valid.")
	case errors.Is(err, state.ErrForbidden):
		writeError(w, http.StatusForbidden, "Akses ditolak: sesi ujian sedang terkunci oleh sistem anti-curang.")
	case errors.Is(err, state.ErrUnavailable):
		writeError(w, http.StatusServiceUnavailable, "Data pendukung belum tersedia di server.")
	default:
		log.Printf("state mutation failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan data.")
	}
}

func (s *Server) handleGetState(w http.ResponseWriter, r *http.Request) {
	authValue, ok := requireUser(w, r)
	if !ok {
		return
	}

	access, scoped, linked := scopeFor(authValue.user)
	if scoped && !linked {
		writeError(w, http.StatusForbidden, "Akun belum terhubung ke profil siswa.")
		return
	}

	data, err := s.storeDB.GetState(r.Context())
	if err != nil {
		log.Printf("read state failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal membaca state.")
		return
	}
	if data == nil {
		writeJSON(w, http.StatusOK, map[string]any{"state": nil})
		return
	}

	if !scoped {
		document, err := decodeState(data)
		if err != nil {
			log.Printf("decode state for staff failed: %v", err)
			writeError(w, http.StatusInternalServerError, "Format state tidak valid.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"state": state.StaffView(document, authValue.user.Role),
		})
		return
	}

	document, err := decodeState(data)
	if err != nil {
		log.Printf("decode state failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Format state tidak valid.")
		return
	}

	view := state.ScopedView(document, access)
	if students, ok := view["students"].([]any); !ok || len(students) == 0 {
		writeError(w, http.StatusForbidden, "Profil siswa tidak ditemukan pada data sekolah.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"state": view})
}

func (s *Server) handlePutState(w http.ResponseWriter, r *http.Request) {
	authValue, ok := requireUser(w, r)
	if !ok {
		return
	}

	writes, err := s.cache.IncrementCounter(r.Context(), "ratelimit:state:"+authValue.user.ID, time.Minute)
	if err != nil {
		log.Printf("state rate limit failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server.")
		return
	}
	if writes > maxStateWritesPerMinute {
		writeError(w, http.StatusTooManyRequests, "Terlalu banyak penyimpanan data. Tunggu sebentar lalu coba lagi.")
		return
	}

	access, scoped, linked := scopeFor(authValue.user)
	if !scoped {
		switch authValue.user.Role {
		case "admin_tu", "headmaster":
		default:
			writeError(w, http.StatusForbidden, "Hanya Admin TU atau Kepala Sekolah yang dapat memperbarui seluruh data.")
			return
		}
	}
	if scoped && !linked {
		writeError(w, http.StatusForbidden, "Akun belum terhubung ke profil siswa.")
		return
	}

	var payload struct {
		State json.RawMessage `json:"state"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if len(payload.State) == 0 || string(payload.State) == "null" {
		writeError(w, http.StatusBadRequest, "State harus berupa objek.")
		return
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload.State, &object); err != nil || object == nil {
		writeError(w, http.StatusBadRequest, "State harus berupa objek.")
		return
	}
	if len(object) == 0 {
		writeError(w, http.StatusBadRequest, "State tidak boleh kosong.")
		return
	}
	if len(object) > 64 {
		writeError(w, http.StatusBadRequest, "Terlalu banyak field pada state.")
		return
	}
	if len(payload.State) > maxStateBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Ukuran state melebihi batas 2 MB.")
		return
	}

	audit := store.StateAudit{
		ActorID: authValue.user.ID,
		Role:    authValue.user.Role,
		Action:  "api.state.put",
	}

	var mergeErr error
	var saveErr error
	if scoped {
		incoming, err := decodeState(payload.State)
		if err != nil {
			writeError(w, http.StatusBadRequest, "State tidak valid.")
			return
		}

		// Merge inside the row lock so a concurrent payment, quiz, or web form
		// cannot be overwritten by a stale read-modify-write snapshot.
		_, saveErr = s.storeDB.UpdateStateAudited(r.Context(), audit, func(raw json.RawMessage) (json.RawMessage, error) {
			existing, err := decodeState(raw)
			if err != nil {
				return nil, err
			}
			if len(existing) == 0 {
				// A browser snapshot must never become the school's data.
				return nil, errStateNotInitialized
			}
			merged, err := state.MergeOwned(existing, incoming, access)
			if err != nil {
				mergeErr = err
				return nil, err
			}
			updated, err := json.Marshal(merged)
			if err != nil {
				return nil, err
			}
			if len(updated) > maxStateBytes {
				return nil, errStateTooLarge
			}
			return updated, nil
		})
	} else {
		// A full admin snapshot still replaces the row atomically, so it cannot
		// interleave with a server-side mutation.
		_, saveErr = s.storeDB.UpdateStateAudited(r.Context(), audit, func(json.RawMessage) (json.RawMessage, error) {
			return payload.State, nil
		})
	}

	if errors.Is(saveErr, store.ErrNoState) {
		if scoped {
			writeError(w, http.StatusConflict, "Data sekolah belum diinisialisasi di server.")
			return
		}
		// Admin import: create the initial state row.
		saveErr = s.storeDB.PutState(r.Context(), payload.State)
		if saveErr == nil {
			_ = s.storeDB.RecordStateAudit(r.Context(), audit)
		}
	}

	if saveErr != nil {
		switch {
		case mergeErr != nil:
			writeError(w, http.StatusForbidden, "Akun belum terhubung ke profil siswa.")
		case errors.Is(saveErr, errStateNotInitialized):
			writeError(w, http.StatusConflict, "Data sekolah belum diinisialisasi di server.")
		case errors.Is(saveErr, errStateTooLarge):
			writeError(w, http.StatusRequestEntityTooLarge, "Ukuran state melebihi batas 2 MB.")
		default:
			log.Printf("save state failed: %v", saveErr)
			writeError(w, http.StatusInternalServerError, "Gagal menyimpan state.")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "saved",
		"updatedAt": timestampNow(),
	})
}

func (s *Server) handleResetState(w http.ResponseWriter, r *http.Request) {
	authValue, ok := requireRole(w, r, "headmaster", "admin_tu")
	if !ok {
		return
	}
	if err := s.storeDB.PutState(r.Context(), state.DefaultStateBytes()); err != nil {
		log.Printf("reset state failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mereset state.")
		return
	}
	if err := s.storeDB.RecordStateAudit(r.Context(), store.StateAudit{
		ActorID: authValue.user.ID, Role: authValue.user.Role, Action: "state.reset",
	}); err != nil {
		log.Printf("state audit failed: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}
