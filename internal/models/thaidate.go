package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var thaiDateRe = regexp.MustCompile(`(\d+)\s*([^\d\s]+)\s*(\d+)`)

// Abbreviated (dots stripped) and full Thai month names.
var thaiMonths = map[string]int{
	"มค": 1, "กพ": 2, "มีค": 3, "เมย": 4, "พค": 5, "มิย": 6,
	"กค": 7, "สค": 8, "กย": 9, "ตค": 10, "พย": 11, "ธค": 12,
	"มกราคม": 1, "กุมภาพันธ์": 2, "มีนาคม": 3, "เมษายน": 4, "พฤษภาคม": 5, "มิถุนายน": 6,
	"กรกฎาคม": 7, "สิงหาคม": 8, "กันยายน": 9, "ตุลาคม": 10, "พฤศจิกายน": 11, "ธันวาคม": 12,
}

// ParseThaiDate converts Thai dates such as "2 มี.ค. 68" or "วันอังคารที่ 20 ตุลาคม 2569"
// to "YYYY-MM-DD" (Gregorian). Returns "" when the input is not recognised.
func ParseThaiDate(s string) string {
	m := thaiDateRe.FindStringSubmatch(s)
	if len(m) < 4 {
		return ""
	}
	month, ok := thaiMonths[strings.ReplaceAll(m[2], ".", "")]
	if !ok {
		return ""
	}
	day, _ := strconv.Atoi(m[1])
	yearBE, _ := strconv.Atoi(m[3])
	if yearBE < 100 {
		yearBE += 2500
	}
	return fmt.Sprintf("%d-%02d-%02d", yearBE-543, month, day)
}
