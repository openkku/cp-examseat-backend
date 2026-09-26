package models

import "regexp"

var nonDigits = regexp.MustCompile("[^0-9]+")

// NormalizeStudentID strips every non-digit, e.g. "653380123-4" -> "6533801234".
func NormalizeStudentID(input string) string {
	return nonDigits.ReplaceAllString(input, "")
}
