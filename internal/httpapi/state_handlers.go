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

	dataToSave := []byte(payload.State)
	if scoped {
		incoming, err := decodeState(payload.State)
		if err != nil {
			writeError(w, http.StatusBadRequest, "State tidak valid.")
			return
		}

		existingRaw, err := s.storeDB.GetState(r.Context())
		if err != nil {
			log.Printf("read state for merge failed: %v", err)
			writeError(w, http.StatusInternalServerError, "Gagal menyimpan state.")
			return
		}

		existing := map[string]any{}
		if existingRaw != nil {
			existing, err = decodeState(existingRaw)
			if err != nil {
				log.Printf("decode existing state failed: %v", err)
				writeError(w, http.StatusInternalServerError, "Format state tidak valid.")
				return
			}
		}

		if len(existing) == 0 && !s.cfg.DemoMode {
			// Production must never let a browser snapshot become the school's
			// data. The initial document is seeded from the embedded default.
			writeError(w, http.StatusConflict, "Data sekolah belum diinisialisasi di server.")
			return
		}

		merged, err := state.MergeOwned(existing, incoming, access)
		if err != nil {
			writeError(w, http.StatusForbidden, "Akun belum terhubung ke profil siswa.")
			return
		}
		dataToSave, err = json.Marshal(merged)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Gagal menyimpan state.")
			return
		}
		if len(dataToSave) > maxStateBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "Ukuran state melebihi batas 2 MB.")
			return
		}
	}

	if err := s.storeDB.PutState(r.Context(), dataToSave); err != nil {
		log.Printf("save state failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan state.")
		return
	}
	auditActor, auditRole := "", ""
	if auth, ok := requestAuthFromContext(r.Context()); ok {
		auditActor, auditRole = auth.user.ID, auth.user.Role
	}
	if err := s.storeDB.RecordStateAudit(r.Context(), store.StateAudit{
		ActorID: auditActor, Role: auditRole, Action: "api.state.put",
	}); err != nil {
		log.Printf("state audit failed: %v", err)
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
