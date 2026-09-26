package models

import (
	"strings"
	"testing"
)

func FuzzNormalizeStudentID(f *testing.F) {
	for _, seed := range []string{"653380123-4", "", "abc", "٦٥٣", "1' OR '1'='1"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		out := NormalizeStudentID(input)
		if strings.Trim(out, "0123456789") != "" {
			t.Fatalf("NormalizeStudentID(%q) = %q contains non-ASCII-digits", input, out)
		}
	})
}

func FuzzParseLabels(f *testing.F) {
	for _, seed := range []string{"", "LAB, Lab, LAB", " , ,", "นัดสอบนอกตาราง"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		seen := map[string]bool{}
		for _, l := range ParseLabels(raw) {
			if l == "" || l != strings.TrimSpace(l) || seen[l] {
				t.Fatalf("ParseLabels(%q) produced bad label %q", raw, l)
			}
			seen[l] = true
		}
	})
}

func FuzzCompareRounds(f *testing.F) {
	f.Add("mid_1_2569", "final_2_2568")
	f.Add("", "_ _ _")
	f.Fuzz(func(t *testing.T, a, b string) {
		// A strict ordering: never both a<b and b<a.
		if CompareRounds(a, b) && CompareRounds(b, a) {
			t.Fatalf("CompareRounds is not antisymmetric for %q, %q", a, b)
		}
	})
}
