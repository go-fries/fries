package cache

import (
	"context"
	"time"

	"github.com/go-fries/fries/v3/locker"
)

type NullStore struct{}

var _ Store = (*NullStore)(nil)

func (n *NullStore) Has(context.Context, string) (bool, error) {
	return true, nil
}

func (n *NullStore) Get(context.Context, string, any) error {
	return nil
}

func (n *NullStore) Put(context.Context, string, any, time.Duration) (bool, error) {
	return true, nil
}

func (n *NullStore) Increment(context.Context, string, int) (int, error) {
	return 0, nil
}

func (n *NullStore) Decrement(context.Context, string, int) (int, error) {
	return 0, nil
}

func (n *NullStore) Forever(context.Context, string, any) (bool, error) {
	return true, nil
}

func (n *NullStore) Forget(context.Context, string) (bool, error) {
	return true, nil
}

func (n *NullStore) Flush(context.Context) (bool, error) {
	return true, nil
}

func (n *NullStore) Lock(string, time.Duration) locker.Locker {
	return nil
}

func (n *NullStore) GetPrefix() string {
	return ""
}
