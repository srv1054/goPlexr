package util

import (
	"fmt"
	"strconv"
)

// CommaInt formats an integer with commas as thousands separators.
func CommaInt(n int) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return sign + s
}

// CommaInt64 formats an int64 with commas as thousands separators.
func CommaInt64(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return sign + s
}

// CommaAny formats an integer (int or int64) with commas, or falls back to fmt.Sprint for other types.
func CommaAny(v any) string {
	switch t := v.(type) {
	case int:
		return CommaInt(t)
	case int64:
		return CommaInt64(t)
	default:
		return fmt.Sprint(v)
	}
}

// BytesHuman formats a byte count into a human-readable string (e.g., 1.5 GiB).
func BytesHuman(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	const unit = 1024
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
