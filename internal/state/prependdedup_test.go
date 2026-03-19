package state

import (
	"testing"
)

func TestPrependDedup(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		max   int
		want  []string
	}{
		{"empty slice", nil, "a", 3, []string{"a"}},
		{"no dup, within cap", []string{"b", "c"}, "a", 5, []string{"a", "b", "c"}},
		{"dedup moves to front", []string{"a", "b", "c"}, "c", 5, []string{"c", "a", "b"}},
		{"trim to max", []string{"b", "c", "d"}, "a", 3, []string{"a", "b", "c"}},
		{"dedup + trim", []string{"a", "b", "c", "d"}, "c", 3, []string{"c", "a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prependDedup(tt.slice, tt.item, tt.max)
			if len(got) != len(tt.want) {
				t.Fatalf("len: got %d %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
