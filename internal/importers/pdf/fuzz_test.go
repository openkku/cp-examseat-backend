package pdf_test

import (
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/importers/pdf"
)

// Roster PDFs come from outside; malformed text must return an error, not panic.
func FuzzParsePDFText(f *testing.F) {
	f.Add("รายวิชา CP421024 OOP\nกลุ่มที่ 1\nห้องสอบ CP9421\nวันเวลาสอบ 22 มี.ค 69 (17:00 – 19:00 น.)\n683380010-2 CP-Cy CP9421-67")
	f.Add("")
	f.Add("ห้องสอบ (((\nวันเวลาสอบ 99 xx 99 ()")
	f.Fuzz(func(t *testing.T, text string) {
		_, _ = pdf.ParsePDFText(text, "r", nil, "", "")
	})
}
