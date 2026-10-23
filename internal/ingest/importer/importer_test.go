package importer

import (
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/ingest"
)

func TestRegistryUnsupportedFile(t *testing.T) {
	if (Registry{}).Supports(".txt") {
		t.Error(".txt must not be supported")
	}
	if _, err := (Registry{}).Extract("test.txt", ingest.Params{RoundID: "test_round"}); err == nil {
		t.Errorf("Expected error for unsupported extension .txt, got nil")
	}
	for _, ext := range []string{".pdf", ".xlsx", ".xls", ".json"} {
		if !(Registry{}).Supports(ext) {
			t.Errorf("%s should be supported", ext)
		}
	}
}
