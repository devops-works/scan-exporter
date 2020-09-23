package scan

import (
	"reflect"
	"testing"
	"time"
)

// func TestTarget_getAddress(t *testing.T) {
// 	type fields struct {
// 		name        string
// 		period      string
// 		ip          string
// 		tcp         protocol
// 		udp         protocol
// 		icmp        protocol
// 		logger      zerolog.Logger
// 		portsToScan map[string][]string
// 		portsOpen   map[string][]string
// 	}
// 	type args struct {
// 		port string
// 	}
// 	tests := []struct {
// 		name   string
// 		fields fields
// 		args   args
// 		want   string
// 	}{
// 		{
// 			name: "basicTest",
// 			fields: fields{
// 				ip: "1.2.3.4",
// 			},
// 			args: args{
// 				port: "80",
// 			},
// 			want: "1.2.3.4:80",
// 		},
// 		{
// 			name: "basicTest2",
// 			fields: fields{
// 				ip: "111.222.222.111",
// 			},
// 			args: args{
// 				port: "666",
// 			},
// 			want: "111.222.222.111:666",
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			tar := &Target{
// 				name:        tt.fields.name,
// 				period:      tt.fields.period,
// 				ip:          tt.fields.ip,
// 				tcp:         tt.fields.tcp,
// 				udp:         tt.fields.udp,
// 				icmp:        tt.fields.icmp,
// 				logger:      tt.fields.logger,
// 				portsToScan: tt.fields.portsToScan,
// 				portsOpen:   tt.fields.portsOpen,
// 			}
// 			if got := tar.getAddress(tt.args.port); got != tt.want {
// 				t.Errorf("Target.getAddress() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestTarget_Name(t *testing.T) {
// 	type fields struct {
// 		name        string
// 		period      string
// 		ip          string
// 		tcp         protocol
// 		udp         protocol
// 		icmp        protocol
// 		logger      zerolog.Logger
// 		portsToScan map[string][]string
// 		portsOpen   map[string][]string
// 	}
// 	tests := []struct {
// 		name   string
// 		fields fields
// 		want   string
// 	}{
// 		{
// 			name: "basicTest",
// 			fields: fields{
// 				name: "localhost",
// 			},
// 			want: "localhost",
// 		},
// 		{
// 			name: "basicTest2",
// 			fields: fields{
// 				name: "best-app-ever",
// 			},
// 			want: "best-app-ever",
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			tar := &Target{
// 				name:        tt.fields.name,
// 				period:      tt.fields.period,
// 				ip:          tt.fields.ip,
// 				tcp:         tt.fields.tcp,
// 				udp:         tt.fields.udp,
// 				icmp:        tt.fields.icmp,
// 				logger:      tt.fields.logger,
// 				portsToScan: tt.fields.portsToScan,
// 				portsOpen:   tt.fields.portsOpen,
// 			}
// 			if got := tar.Name(); got != tt.want {
// 				t.Errorf("Target.Name() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func Test_protocol_getDuration(t *testing.T) {
// 	type fields struct {
// 		period   string
// 		rng      string
// 		expected string
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		want    time.Duration
// 		wantErr bool
// 	}{
// 		{
// 			name: "seconds",
// 			fields: fields{
// 				period: "60s",
// 			},
// 			want:    60 * time.Second,
// 			wantErr: false,
// 		},
// 		{
// 			name: "minutes",
// 			fields: fields{
// 				period: "10m",
// 			},
// 			want:    10 * time.Minute,
// 			wantErr: false,
// 		},
// 		{
// 			name: "hours",
// 			fields: fields{
// 				period: "42h",
// 			},
// 			want:    42 * time.Hour,
// 			wantErr: false,
// 		},
// 		{
// 			name: "days",
// 			fields: fields{
// 				period: "13d",
// 			},
// 			want:    13 * 24 * time.Hour,
// 			wantErr: false,
// 		},
// 		{
// 			name: "fail",
// 			fields: fields{
// 				period: "14f",
// 			},
// 			want:    0,
// 			wantErr: true,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			p := &protocol{
// 				period:   tt.fields.period,
// 				rng:      tt.fields.rng,
// 				expected: tt.fields.expected,
// 			}
// 			got, err := p.getDuration()
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("protocol.getDuration() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if got != tt.want {
// 				t.Errorf("protocol.getDuration() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

func Test_stringInSlice(t *testing.T) {
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
			if got := stringInSlice(tt.s, tt.sl); got != tt.want {
				t.Errorf("stringInSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readPortsRange(t *testing.T) {
	tests := []struct {
		name    string
		ranges  string
		want    []string
		wantErr bool
	}{
		{name: "empty range", ranges: "", want: []string{}, wantErr: false},
		{name: "simple port", ranges: "122", want: []string{"122"}, wantErr: false},
		{name: "port range", ranges: "122-125", want: []string{"122", "123", "124", "125"}, wantErr: false},
		{name: "equal range", ranges: "122-122", want: []string{"122"}, wantErr: false},
		{name: "multiple ports", ranges: "122,123,124", want: []string{"122", "123", "124"}, wantErr: false},
		{name: "multiple ranges", ranges: "122-125,130-131", want: []string{"122", "123", "124", "125", "130", "131"}, wantErr: false},
		{name: "mixed ranges", ranges: "122-125,131", want: []string{"122", "123", "124", "125", "131"}, wantErr: false},
		{name: "mixed ranges", ranges: "120,122-125", want: []string{"120", "122", "123", "124", "125"}, wantErr: false},
		{name: "mixed ranges", ranges: "120,122-125,131", want: []string{"120", "122", "123", "124", "125", "131"}, wantErr: false},
		{name: "inverted range", ranges: "500-400", wantErr: true},
		{name: "overflow range", ranges: "6500-65536", wantErr: true},
		{name: "missing first item in range", ranges: "-123", wantErr: true},
		{name: "missing last item in range", ranges: "123-", wantErr: true},
		{name: "error", ranges: "a-b", wantErr: true},
		{name: "error", ranges: "a", wantErr: true},
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

func TestTarget_createJobs(t *testing.T) {
	tests := []struct {
		name         string
		pts          map[string][]string
		workersCount int
		wantErr      bool
	}{
		{name: "5-1", pts: map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}}, workersCount: 1},
		{name: "5-2", pts: map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}}, workersCount: 2},
		{name: "5-3", pts: map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}}, workersCount: 3},
		{name: "5-4", pts: map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}}, workersCount: 4},
		{name: "5-5", pts: map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}}, workersCount: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := &Target{
				portsToScan: tt.pts,
			}
			workersCount = tt.workersCount
			got, err := tg.createJobs("tcp")
			if (err != nil) != tt.wantErr {
				t.Errorf("Target.createJobs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.workersCount {
				t.Errorf("Target.createJobs() = %d, wanted %d jobs; joblist %v", len(got), tt.workersCount, got)
			}
		})
	}
}

func Test_getDuration(t *testing.T) {
	tests := []struct {
		name    string
		period  string
		want    time.Duration
		wantErr bool
	}{
		{name: "seconds", period: "666s", want: 666 * time.Second, wantErr: false},
		{name: "minutes", period: "42m", want: 42 * time.Minute, wantErr: false},
		{name: "hours", period: "69h", want: 69 * time.Hour, wantErr: false},
		{name: "days", period: "13d", want: 13 * 24 * time.Hour, wantErr: false},
		{name: "error", period: "1337gg", wantErr: true},
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
