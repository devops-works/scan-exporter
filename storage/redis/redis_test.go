package redis

import (
	"log"
	"reflect"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
)

func TestInstance_ReplaceList(t *testing.T) {
	tests := []struct {
		name    string
		setName string
		items   []string
		wantErr bool
	}{
		{name: "test1", setName: "super-name", items: []string{"item1", "item2", "item3"}, wantErr: false},
	}
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Instance{
				rdb: client,
			}
			if err := i.ReplaceList(tt.setName, tt.items); (err != nil) != tt.wantErr {
				t.Errorf("Instance.ReplaceList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInstance_ReadList(t *testing.T) {
	tests := []struct {
		name        string
		setName     string
		want        []string
		dataToWrite []string
		wantErr     bool
	}{
		{name: "test1", setName: "super-name", dataToWrite: []string{"item1", "item2"}, want: []string{"item1", "item2"}, wantErr: false},
		{name: "test2", setName: "super-name", dataToWrite: []string{"some", "thing", "else"}, want: []string{"else", "some", "thing"}, wantErr: false},
	}

	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	for _, tt := range tests {
		client.Del(tt.setName)
		client.SAdd(tt.setName, tt.dataToWrite)
		t.Run(tt.name, func(t *testing.T) {
			i := &Instance{
				rdb: client,
			}
			got, err := i.ReadList(tt.setName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Instance.ReadList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Instance.ReadList() = %v, want %v", got, tt.want)
			}
		})
	}
}
