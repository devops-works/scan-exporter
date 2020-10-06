package common

import "testing"

func Test_StringInSlice(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sl   []string
		want bool
	}{
		{name: "exists", s: "a", sl: []string{"a", "b", "c"}, want: true},
		{name: "does not exist", s: "z", sl: []string{"a", "b", "c"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringInSlice(tt.s, tt.sl); got != tt.want {
				t.Errorf("stringInSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
