package models

import "testing"

func TestParseThaiDate(t *testing.T) {
	cases := map[string]string{
		"2 มี.ค. 68":                    "2025-03-02",
		"23 มี.ค. 69 (เช้า)":            "2026-03-23",
		"กักตัวสอบ 26 ส.ค. 69 (ช.)":     "2026-08-26",
		"วันอังคารที่ 20 ตุลาคม 2569":   "2026-10-20",
		"📅 วันจันทร์ที่ 19 ตุลาคม 2569": "2026-10-19",
		"วันศุกร์ที่ 1 มกราคม 2570":     "2027-01-01",
		"รายวิชากักตัวสอบ":              "",
		"20 octember 2569": "",
	}
	for in, want := range cases {
		if got := ParseThaiDate(in); got != want {
			t.Errorf("ParseThaiDate(%q) = %q, want %q", in, got, want)
		}
	}
}
