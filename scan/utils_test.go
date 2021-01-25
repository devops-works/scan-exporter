package scan

import (
	"reflect"
	"testing"
	"time"
)

func Test_getDuration(t *testing.T) {
	tests := []struct {
		name    string
		period  string
		want    time.Duration
		wantErr bool
	}{
		{name: "seconds", period: "42s", want: 42 * time.Second, wantErr: false},
		{name: "minutes", period: "666m", want: 666 * time.Minute, wantErr: false},
		{name: "hours", period: "1337h", want: 1337 * time.Hour, wantErr: false},
		{name: "days", period: "69d", want: 69 * 24 * time.Hour, wantErr: false},
		{name: "error", period: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDuration(tt.period)
			if (err != nil) != tt.wantErr {
				t.Errorf("getDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readPortsRange(t *testing.T) {
	tests := []struct {
		name    string
		ranges  string
		want    []int
		wantErr bool
	}{
		{name: "single port", ranges: "1", want: []int{1}, wantErr: false},
		{name: "comma", ranges: "1,22", want: []int{1, 22}, wantErr: false},
		{name: "hyphen", ranges: "22-25", want: []int{22, 23, 24, 25}, wantErr: false},
		{name: "comma and hyphen", ranges: "22,30-32", want: []int{22, 30, 31, 32}, wantErr: false},
		{name: "comma, hyphen, comma", ranges: "22,30-32,50", want: []int{22, 30, 31, 32, 50}, wantErr: false},
		{name: "unknown", ranges: "foobar", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readPortsRange(tt.ranges)
			if (err != nil) != tt.wantErr {
				t.Errorf("readPortsRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readPortsRange() = %v, want %v", got, tt.want)
			}
		})
	}
}
