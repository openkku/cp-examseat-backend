package models

import (
	"sort"
	"testing"
)

func TestCompareRounds(t *testing.T) {
	rounds := []string{
		"final_1_2567",
		"mid_1_2569",
		"mid_2_2568",
		"final_2_2568",
		"final_1_2568",
	}

	expected := []string{
		"mid_1_2569",
		"final_2_2568",
		"mid_2_2568",
		"final_1_2568",
		"final_1_2567",
	}

	sort.Slice(rounds, func(i, j int) bool {
		return CompareRounds(rounds[i], rounds[j])
	})

	for i, r := range rounds {
		if r != expected[i] {
			t.Errorf("At index %d: expected %s, got %s", i, expected[i], r)
		}
	}
}

func TestParseLabels(t *testing.T) {
	tests := []struct {
		raw      string
		expected []string
	}{
		{"", nil},
		{"Lecture", []string{"Lecture"}},
		{"LAB, Lab, LAB", []string{"LAB", "Lab"}},
		{" นัดสอบนอกตาราง , Lecture ", []string{"นัดสอบนอกตาราง", "Lecture"}},
	}

	for _, tt := range tests {
		result := ParseLabels(tt.raw)
		if len(result) != len(tt.expected) {
			t.Errorf("ParseLabels(%q) expected %v, got %v", tt.raw, tt.expected, result)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("ParseLabels(%q)[%d] expected %s, got %s", tt.raw, i, tt.expected[i], result[i])
			}
		}
	}
}

func TestNormalizeStudentID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"683380001-1", "6833800011"},
		{" 653380123-4 ", "6533801234"},
		{"", ""},
	}

	for _, tt := range tests {
		res := NormalizeStudentID(tt.input)
		if res != tt.expected {
			t.Errorf("NormalizeStudentID(%q) expected %q, got %q", tt.input, tt.expected, res)
		}
	}
}

func TestDefaultRoundLabel(t *testing.T) {
	tests := map[string]string{
		"mid_1_2569":    "กลางภาค 1/2569",
		"final_2_2568":  "ปลายภาค 2/2568",
		"2_2568":        "2_2568",
		"summer_3_2568": "summer_3_2568",
	}
	for id, want := range tests {
		if got := DefaultRoundLabel(id); got != want {
			t.Errorf("DefaultRoundLabel(%q) = %q; want %q", id, got, want)
		}
	}
}

func TestResolveCategory(t *testing.T) {
	tests := []struct {
		seat Seat
		want string
	}{
		{Seat{}, CategoryInSchedule},
		{Seat{Category: "CUSTOM"}, "CUSTOM"},
		{Seat{CustomID: "LAB"}, CategoryOutOfSchedule},
		{Seat{Labels: []string{"Lecture", LabelOutOfSchedule}}, CategoryOutOfSchedule},
	}
	for _, tc := range tests {
		if got := tc.seat.ResolveCategory(); got != tc.want {
			t.Errorf("ResolveCategory(%+v) = %q; want %q", tc.seat, got, tc.want)
		}
	}
}

func TestSplitTime(t *testing.T) {
	tests := []struct{ in, start, end string }{
		{"09.00 - 12.00", "09.00", "12.00"},
		{"13.00", "13.00", ""},
		{"", "", ""},
	}
	for _, tc := range tests {
		start, end := SplitTime(tc.in)
		if start != tc.start || end != tc.end {
			t.Errorf("SplitTime(%q) = %q, %q; want %q, %q", tc.in, start, end, tc.start, tc.end)
		}
	}
}
