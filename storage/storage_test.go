package storage

import (
	"testing"
)

func TestStore_Get(t *testing.T) {
	tests := []struct {
		name     string
		s        Store
		k        string
		expected []string
	}{
		{
			name:     "test",
			s:        map[string][]string{"foo": {"bar"}, "toor": {"root"}},
			k:        "toor",
			expected: []string{"root"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.s.Get(tt.k)
			if !equal(output, tt.expected) {
				t.Errorf("got %q want %q given, %q", output, tt.expected, "test")
			}
		})
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
