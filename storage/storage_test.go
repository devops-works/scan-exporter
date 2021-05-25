package storage

import (
	"reflect"
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
				t.Errorf("got %q want %q given", output, tt.expected)
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
func TestStore_Add(t *testing.T) {
	tests := []struct {
		name     string
		s        Store
		k        string
		v        string
		expected []string
	}{
		{
			name:     "add value to non-existing key",
			s:        map[string][]string{"toor": {"root"}},
			k:        "foo",
			v:        "bar",
			expected: []string{"bar"},
		},
		{
			name:     "add value to existing key",
			s:        map[string][]string{"toor": {"root"}},
			k:        "toor",
			v:        "bar",
			expected: []string{"root", "bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.Add(tt.k, tt.v)
			if !equal(tt.s[tt.k], tt.expected) {
				t.Errorf("got %q want %q given", tt.s[tt.k], tt.expected)
			}
		})
	}
}

func TestStore_Delete(t *testing.T) {
	tests := []struct {
		name     string
		s        Store
		k        string
		v        string
		expected Store
	}{
		{
			name:     "remove key",
			s:        map[string][]string{"toor": {"root"}},
			k:        "toor",
			expected: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.Delete(tt.k)
			if len(tt.s) != len(tt.expected) {
				t.Errorf("got %q want %q given", tt.s, tt.expected)
			}
		})
	}
}

func TestStore_Update(t *testing.T) {
	tests := []struct {
		name     string
		s        Store
		k        string
		v        []string
		expected []string
	}{
		{
			name:     "update key",
			s:        map[string][]string{"toor": {"root"}},
			k:        "toor",
			v:        []string{"bar"},
			expected: []string{"bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.s.Update(tt.k, tt.v)
			if !equal(tt.s[tt.k], tt.expected) {
				t.Errorf("got %q want %q given", tt.s[tt.k], tt.expected)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name string
		want Store
	}{
		{
			name: "create basic store",
			want: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Create(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Create() = %v, want %v", got, tt.want)
			}
		})
	}
}
