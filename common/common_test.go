package common

import (
	"testing"
)

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

func TestCompareStringSlices(t *testing.T) {

	tests := []struct {
		name string
		sl1  []string
		sl2  []string
		want int
	}{
		{
			name: "same lists", sl1: []string{"1", "2", "3"}, sl2: []string{"1", "2", "3"}, want: 0,
		},
		{
			name: "different length1", sl1: []string{"1", "2", "3"}, sl2: []string{"1", "2", "3", "4"}, want: 1,
		},
		{
			name: "different length2", sl1: []string{"1", "2", "3", "4"}, sl2: []string{"1", "2", "3"}, want: 1,
		},
		{
			name: "same content, not same order", sl1: []string{"1", "2", "3"}, sl2: []string{"3", "2", "1"}, want: 0,
		},
		{
			name: "different lists1", sl1: []string{"1", "2", "3"}, sl2: []string{"4", "5", "6"}, want: 6,
		},
		{
			name: "different lists2", sl1: []string{"0", "1", "2", "3"}, sl2: []string{"4", "5", "6"}, want: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareStringSlices(tt.sl1, tt.sl2); got != tt.want {
				t.Errorf("CompareStringSlices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkCompareStringSlices(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CompareStringSlices([]string{"0", "1", "2", "3"}, []string{"4", "5", "6"})
	}
}
