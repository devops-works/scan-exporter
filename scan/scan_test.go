package scan

import (
	"reflect"
	"testing"
	"time"
)

func TestTarget_setPorts(t *testing.T) {
	type args struct {
		proto  string
		period string
		rng    string
		exp    string
	}
	tests := []struct {
		name        string
		protos      map[string]protocol
		portsToScan map[string][]string
		args        args
		wantErr     bool
	}{
		{
			name:        "test1",
			args:        args{proto: "tcp", period: "1s", rng: "reserved", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     false,
		},
		{
			name:        "test2",
			args:        args{proto: "tcp", period: "1s", rng: "blah", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     true,
		},
		{
			name:        "test3",
			args:        args{proto: "not-a-proto", period: "1s", rng: "all", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     true,
		},
		{
			name:        "test4",
			args:        args{proto: "udp", period: "1s", rng: "18-796", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     false,
		},
		{
			name:        "test5",
			args:        args{proto: "udp", period: "1s", rng: "42-13", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     true,
		},
		{
			name:        "test6",
			args:        args{proto: "udp", period: "1s", rng: "13-15,42-666", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     false,
		},
		{
			name:        "test7",
			args:        args{proto: "tcp", period: "1s", rng: "13,42-666,1337", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     false,
		},
		{
			name:        "test8",
			args:        args{proto: "tcp", period: "1s", rng: "13,42,666,1337", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     false,
		},
		{
			name:        "test9",
			args:        args{proto: "tcp", period: "1s", rng: "", exp: "666"},
			protos:      make(map[string]protocol),
			portsToScan: make(map[string][]string),
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tar := &Target{
				protos:      tt.protos,
				portsToScan: tt.portsToScan,
			}
			if err := tar.setPorts(tt.args.proto, tt.args.period, tt.args.rng, tt.args.exp); (err != nil) != tt.wantErr {
				t.Errorf("Target.setPorts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTarget_Name(t *testing.T) {
	tests := []struct {
		name  string
		tName string
		want  string
	}{
		{name: "test", tName: "myApp", want: "myApp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tar := &Target{
				name: tt.tName,
			}
			if got := tar.Name(); got != tt.want {
				t.Errorf("Target.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_checkAccordance(t *testing.T) {
	type fields struct {
		protos map[string]protocol
	}
	type args struct {
		proto string
		open  []string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		unexpected []string
		closed     []string
		wantErr    bool
	}{
		{
			name:       "test1",
			fields:     fields{protos: map[string]protocol{"tcp": protocol{expected: "1337"}}},
			args:       args{proto: "tcp", open: []string{"1337"}},
			unexpected: []string{},
			closed:     []string{},
			wantErr:    false,
		},
		{
			name:       "test2",
			fields:     fields{protos: map[string]protocol{"tcp": protocol{expected: "1337"}}},
			args:       args{proto: "tcp", open: []string{"666"}},
			unexpected: []string{"666"},
			closed:     []string{"1337"},
			wantErr:    false,
		},
		{
			name:       "test3",
			fields:     fields{protos: map[string]protocol{"tcp": protocol{expected: "11,22,33"}}},
			args:       args{proto: "tcp", open: []string{"22"}},
			unexpected: []string{},
			closed:     []string{"11", "33"},
			wantErr:    false,
		},
		{
			name:       "test4",
			fields:     fields{protos: map[string]protocol{"tcp": protocol{expected: "11,22,33"}}},
			args:       args{proto: "tcp", open: []string{"44"}},
			unexpected: []string{"44"},
			closed:     []string{"11", "22", "33"},
			wantErr:    false,
		},
		{
			name:       "test5",
			fields:     fields{protos: map[string]protocol{"tcp": protocol{expected: "11-15"}}},
			args:       args{proto: "tcp", open: []string{"11", "12", "13", "14", "15"}},
			unexpected: []string{},
			closed:     []string{},
			wantErr:    false,
		},
		{
			name:       "test6",
			fields:     fields{protos: map[string]protocol{"tcp": protocol{expected: "11-15"}}},
			args:       args{proto: "tcp", open: []string{"11", "12", "13", "14"}},
			unexpected: []string{},
			closed:     []string{"15"},
			wantErr:    false,
		},
		{
			name:       "test7",
			fields:     fields{protos: map[string]protocol{"tcp": protocol{expected: "11-15"}}},
			args:       args{proto: "tcp", open: []string{"11", "12", "13", "14", "16"}},
			unexpected: []string{"16"},
			closed:     []string{"15"},
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tar := &Target{
				protos: tt.fields.protos,
			}
			got, got1, err := tar.checkAccordance(tt.args.proto, tt.args.open)
			if (err != nil) != tt.wantErr {
				t.Errorf("Target.checkAccordance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.unexpected) {
				t.Errorf("Target.checkAccordance() got = %v, want %v", got, tt.unexpected)
			}
			if !reflect.DeepEqual(got1, tt.closed) {
				t.Errorf("Target.checkAccordance() got1 = %v, want %v", got1, tt.closed)
			}
		})
	}
}

func TestTarget_getWantedProto(t *testing.T) {
	tests := []struct {
		name   string
		protos map[string]protocol
		want   []string
	}{
		{
			name:   "tcp",
			protos: map[string]protocol{"tcp": protocol{period: "1s"}},
			want:   []string{"tcp"},
		},
		{
			name:   "udp",
			protos: map[string]protocol{"udp": protocol{period: "1h"}},
			want:   []string{"udp"},
		},
		{
			name:   "icmp",
			protos: map[string]protocol{"icmp": protocol{period: "10m"}},
			want:   []string{"icmp"},
		},
		{
			name: "tcp/icmp",
			protos: map[string]protocol{
				"tcp":  protocol{period: "1s"},
				"icmp": protocol{period: "10s"},
			},
			want: []string{"tcp", "icmp"},
		},
		{
			name: "tcp/udp/icmp",
			protos: map[string]protocol{
				"tcp":  protocol{period: "1s"},
				"udp":  protocol{period: "6h"},
				"icmp": protocol{period: "10s"},
			},
			want: []string{"tcp", "udp", "icmp"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tar := &Target{
				protos: tt.protos,
			}
			if got := tar.getWantedProto(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Target.getWantedProto() = %v, want %v", got, tt.want)
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
		{
			name:         "5-1",
			pts:          map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}},
			workersCount: 1,
		},
		{
			name:         "5-2",
			pts:          map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}},
			workersCount: 2,
		},
		{
			name:         "5-3",
			pts:          map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}},
			workersCount: 3,
		},
		{
			name:         "5-4",
			pts:          map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}},
			workersCount: 4,
		},
		{
			name:         "5-5",
			pts:          map[string][]string{"tcp": []string{"1", "2", "3", "4", "5"}},
			workersCount: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := &Target{
				portsToScan: tt.pts,
				workers:     tt.workersCount,
			}
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
