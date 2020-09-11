package scan

import (
	"reflect"
	"testing"
)

func Test_readRange(t *testing.T) {
	type args struct {
		portsRange string
	}
	tests := []struct {
		name    string
		args    args
		want    []int
		wantErr bool
	}{
		{
			name: "simple value",
			args: args{
				portsRange: "12",
			},
			want:    []int{12},
			wantErr: false,
		},
		{
			name: "single coma",
			args: args{
				portsRange: "12,23",
			},
			want:    []int{12, 23},
			wantErr: false,
		},
		{
			name: "single dash",
			args: args{
				portsRange: "12-14",
			},
			want:    []int{12, 13, 14},
			wantErr: false,
		},
		{
			name: "multiple comas",
			args: args{
				portsRange: "12,13,14",
			},
			want:    []int{12, 13, 14},
			wantErr: false,
		},
		{
			name: "multiple dashes",
			args: args{
				portsRange: "12-14,45-48",
			},
			want:    []int{12, 13, 14, 45, 46, 47, 48},
			wantErr: false,
		},
		{
			name: "comas and dahs",
			args: args{
				portsRange: "12,14-16,48",
			},
			want:    []int{12, 14, 15, 16, 48},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readRange(tt.args.portsRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("readRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readRange() = %v, want %v", got, tt.want)
			}
		})
	}
}
