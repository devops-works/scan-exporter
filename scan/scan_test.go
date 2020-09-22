package scan

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestTarget_getAddress(t *testing.T) {
	type fields struct {
		name        string
		period      string
		ip          string
		tcp         protocol
		udp         protocol
		icmp        protocol
		logger      zerolog.Logger
		portsToScan map[string][]string
		portsOpen   map[string][]string
	}
	type args struct {
		port string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "basicTest",
			fields: fields{
				ip: "1.2.3.4",
			},
			args: args{
				port: "80",
			},
			want: "1.2.3.4:80",
		},
		{
			name: "basicTest2",
			fields: fields{
				ip: "111.222.222.111",
			},
			args: args{
				port: "666",
			},
			want: "111.222.222.111:666",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tar := &Target{
				name:        tt.fields.name,
				period:      tt.fields.period,
				ip:          tt.fields.ip,
				tcp:         tt.fields.tcp,
				udp:         tt.fields.udp,
				icmp:        tt.fields.icmp,
				logger:      tt.fields.logger,
				portsToScan: tt.fields.portsToScan,
				portsOpen:   tt.fields.portsOpen,
			}
			if got := tar.getAddress(tt.args.port); got != tt.want {
				t.Errorf("Target.getAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_Name(t *testing.T) {
	type fields struct {
		name        string
		period      string
		ip          string
		tcp         protocol
		udp         protocol
		icmp        protocol
		logger      zerolog.Logger
		portsToScan map[string][]string
		portsOpen   map[string][]string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "basicTest",
			fields: fields{
				name: "localhost",
			},
			want: "localhost",
		},
		{
			name: "basicTest2",
			fields: fields{
				name: "best-app-ever",
			},
			want: "best-app-ever",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tar := &Target{
				name:        tt.fields.name,
				period:      tt.fields.period,
				ip:          tt.fields.ip,
				tcp:         tt.fields.tcp,
				udp:         tt.fields.udp,
				icmp:        tt.fields.icmp,
				logger:      tt.fields.logger,
				portsToScan: tt.fields.portsToScan,
				portsOpen:   tt.fields.portsOpen,
			}
			if got := tar.Name(); got != tt.want {
				t.Errorf("Target.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_protocol_getDuration(t *testing.T) {
	type fields struct {
		period   string
		rng      string
		expected string
	}
	tests := []struct {
		name    string
		fields  fields
		want    time.Duration
		wantErr bool
	}{
		{
			name: "seconds",
			fields: fields{
				period: "60s",
			},
			want:    60 * time.Second,
			wantErr: false,
		},
		{
			name: "minutes",
			fields: fields{
				period: "10m",
			},
			want:    10 * time.Minute,
			wantErr: false,
		},
		{
			name: "hours",
			fields: fields{
				period: "42h",
			},
			want:    42 * time.Hour,
			wantErr: false,
		},
		{
			name: "days",
			fields: fields{
				period: "13d",
			},
			want:    13 * 24 * time.Hour,
			wantErr: false,
		},
		{
			name: "fail",
			fields: fields{
				period: "14f",
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &protocol{
				period:   tt.fields.period,
				rng:      tt.fields.rng,
				expected: tt.fields.expected,
			}
			got, err := p.getDuration()
			if (err != nil) != tt.wantErr {
				t.Errorf("protocol.getDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("protocol.getDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
