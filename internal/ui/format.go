package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func FormatRupiah(amount int) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	raw := strconv.Itoa(amount)
	var parts []string
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)
	out := "Rp " + strings.Join(parts, ".")
	if negative {
		return "-" + out
	}
	return out
}

func FormatDateID(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.In(time.FixedZone("WIB", 7*60*60)).Format("02 Jan 2006, 15:04 WIB")
		}
	}
	return value
}

// FormatDateTimeID renders time.Time values for the security page.
func FormatDateTimeID(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.In(time.FixedZone("WIB", 7*60*60)).Format("02 Jan 2006, 15:04 WIB")
}

func FormatClockID(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.In(time.FixedZone("WIB", 7*60*60)).Format("15:04 WIB")
	}
	return value
}

func StatusLabel(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PAID":
		return "Lunas"
	case "PARTIAL":
		return "Cicilan"
	case "UNPAID":
		return "Belum dibayar"
	case "HADIR_GERBANG":
		return "Hadir tepat waktu"
	case "SAKIT_ORTU":
		return "Sakit (izin orang tua)"
	case "IZIN_ORTU":
		return "Izin orang tua"
	case "ALPA_BOLOS":
		return "Alpa"
	case "VALID_FULL":
		return "Guru hadir & mengajar penuh"
	case "TASK_ONLY":
		return "Guru hanya memberi tugas"
	case "EMPTY_CLASS":
		return "Kelas kosong"
	case "PENDING":
		return "Menunggu validasi"
	default:
		if status == "" {
			return "—"
		}
		return strings.ReplaceAll(strings.ToLower(status), "_", " ")
	}
}

func StatusTone(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PAID", "HADIR_GERBANG", "VALID_FULL":
		return "success"
	case "PARTIAL", "SAKIT_ORTU", "IZIN_ORTU", "TASK_ONLY", "PENDING":
		return "warning"
	case "UNPAID", "ALPA_BOLOS", "EMPTY_CLASS":
		return "danger"
	default:
		return "neutral"
	}
}

func Pct(value, total int) int {
	if total <= 0 {
		return 0
	}
	pct := int(float64(value) / float64(total) * 100)
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func Decimal(value int) string {
	return fmt.Sprintf("%d", value)
}
