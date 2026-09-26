package pdf_test

import (
	"github.com/openkku/cp-examseat-backend/internal/importers/pdf"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSeatsFromSamplePDF(t *testing.T) {
	pdfPath := filepath.Join("..", "..", "..", "data", "source", "custom", "Lab_2568_CP421024_OOP_CY_FinalExam_Sec1.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: sample PDF not found at %s", pdfPath)
	}

	seats, err := pdf.ExtractSeats(pdfPath, "2_2568", []string{"LAB", "Lab"}, "CP9421_LAB", "FINAL_2_OOP_LAB_2026")
	if err != nil {
		t.Fatalf("Failed to extract seats from PDF: %v", err)
	}

	if len(seats) == 0 {
		t.Fatalf("Expected extracted seats > 0, got 0")
	}

	// Verify sample student
	first := seats[0]
	if first.Subject != "CP421024" {
		t.Errorf("Expected subject CP421024, got %s", first.Subject)
	}
	if first.Room != "CP9421" {
		t.Errorf("Expected room CP9421, got %s", first.Room)
	}
	if first.CustomID != "FINAL_2_OOP_LAB_2026" {
		t.Errorf("Expected custom_id FINAL_2_OOP_LAB_2026, got %s", first.CustomID)
	}
	if len(first.Labels) == 0 {
		t.Errorf("Expected labels slice, got empty")
	}
}
