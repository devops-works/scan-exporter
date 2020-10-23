package redis

import (
	"github.com/go-redis/redis"
)

// Instance of Redis backend
type Instance struct {
	url string
	rdb *redis.Client
}

// New creates a new Redis instance
func New(url string) (*Instance, error) {
	// TODO: Check url
	i := &Instance{
		url: url,
	}

	// TODO: Initialize & test connection
	if err := i.init(); err != nil {
		return nil, err
	}
	return i, nil
}

// ReplaceList writes items in a Redis dataset called setName.
func (i *Instance) ReplaceList(setName string, items []string) error {
	i.rdb.Del(setName)
	for _, item := range items {
		err := i.rdb.SAdd(setName, item).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

// ReadList reads items from a Redis dataset called setName.
func (i *Instance) ReadList(setName string) ([]string, error) {
	items, err := i.rdb.SMembers(setName).Result()
	if err != nil {
		return nil, err
	}
	return items, nil
}

// init initiates a new Redis client item.
func (i *Instance) init() error {
	opt, err := redis.ParseURL(i.url)
	if err != nil {
		return err
	}

	i.rdb = redis.NewClient(opt)

	_, err = i.rdb.Ping().Result()
	if err != nil {
		return err
	}
	return nil
}
