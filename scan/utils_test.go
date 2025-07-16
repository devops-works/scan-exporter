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
	reservedPorts := make([]int, 1023)
	for i := range 1023 {
		reservedPorts[i] = i + 1
	}

	allPorts := make([]int, 65535)
	for i := range 65535 {
		allPorts[i] = i + 1
	}

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

		// Tests for keywords
		{name: "keyword all", ranges: "all", want: allPorts, wantErr: false},
		{name: "keyword reserved", ranges: "reserved", want: reservedPorts, wantErr: false},
		{name: "keyword top1000", ranges: "top1000", want: top1000Ports, wantErr: false},

		// Tests for combinations and uniqueness
		{name: "duplicates", ranges: "80,81,443,79-88", want: []int{79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 443}, wantErr: false},
		{name: "reserved with duplicates", ranges: "1,2,reserved", want: reservedPorts, wantErr: false},
		{name: "all with others", ranges: "80,all,9000", want: allPorts, wantErr: false},

		// Tests for edge cases
		{name: "empty string", ranges: "", want: []int{}, wantErr: false},
		{name: "whitespace and commas", ranges: " , ", want: []int{}, wantErr: false},

		// Tests for error conditions
		{name: "unknown keyword", ranges: "foobar", wantErr: true},
		{name: "invalid range min > max", ranges: "100-20", wantErr: true},
		{name: "port > 65535", ranges: "65536", wantErr: true},
		{name: "port < 1", ranges: "0", wantErr: true},
		{name: "range end > 65535", ranges: "65530-65536", wantErr: true},
		{name: "malformed range end", ranges: "100-", wantErr: true},
		{name: "malformed range start", ranges: "-100", wantErr: true},
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
