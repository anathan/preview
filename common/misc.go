package common

import (
	"strings"
)

// NKG: This is lazy, I know.
func buildIn(count int) string {
	results := make([]string, 0, 0)
	for i := 0; i < count; i++ {
		results = append(results, "?")
	}
	return strings.Join(results, ",")
}
