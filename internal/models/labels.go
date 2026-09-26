package models

import "strings"

// ParseLabels splits a comma-separated string into a slice of clean, unique label strings.
func ParseLabels(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return uniqueTrimmed(strings.Split(raw, ","))
}

// JoinLabels joins a slice of label strings into a single comma-separated string after deduplication.
func JoinLabels(labels []string) string {
	return strings.Join(uniqueTrimmed(labels), ",")
}

// HasLabel reports whether labels contains target, ignoring case.
func HasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, target) {
			return true
		}
	}
	return false
}

func uniqueTrimmed(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			out = append(out, trimmed)
		}
	}
	return out
}
