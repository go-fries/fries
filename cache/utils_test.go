package cache

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type utilMockStore struct {
	NullStore

	data map[string]struct {
		value   any
		expired time.Time
	}
	mu sync.Mutex
}

func newUtilMockStore() *utilMockStore {
	return &utilMockStore{
		data: make(map[string]struct {
			value   any
			expired time.Time
		}),
	}
}

func (t *utilMockStore) Get(_ context.Context, key string, dest any) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if value, ok := t.data[key]; ok {
		if time.Now().After(value.expired) {
			delete(t.data, key)
			return ErrNotFound
		}
		if err := setPointerValue(dest, value.value); err != nil {
			return err
		}
		return nil
	}
	return ErrNotFound
}

func (t *utilMockStore) Put(_ context.Context, key string, value any, ttl time.Duration) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.data[key] = struct {
		value   any
		expired time.Time
	}{
		value:   value,
		expired: time.Now().Add(ttl),
	}

	return true, nil
}

func setPointerValue(dest any, value any) error {
	v := reflect.ValueOf(dest)

	if v.Kind() != reflect.Pointer {
		return fmt.Errorf("dest must be a pointer type")
	}
	elem := v.Elem()

	if !elem.CanSet() {
		return fmt.Errorf("the value pointed to by the pointer cannot be modified")
	}

	val := reflect.ValueOf(value)
	if elem.Type() != val.Type() {
		return fmt.Errorf("type mismatch: pointer type is %v, assigned type is %v", elem.Type(), val.Type())
	}
	elem.Set(val)
	return nil
}

func TestUtils_Get(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository(newUtilMockStore())

	ok, err := repo.Set(ctx, "test_key", "test_value", time.Second*10)
	assert.NoError(t, err)
	assert.True(t, ok)

	value, err := Get[string](ctx, repo, "test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", value)
}

func TestUtils_Remember(t *testing.T) {
	ctx := t.Context()
	repo := NewRepository(newUtilMockStore())
	var total int32

	rememberFunc := func(value string) (string, error) {
		return Remember(ctx, repo, "test_key", time.Millisecond*100, func() (string, error) {
			atomic.AddInt32(&total, 1)
			return value, nil
		})
	}

	assert.Equal(t, int32(0), total)

	v1, err1 := rememberFunc("value1")
	assert.NoError(t, err1)
	assert.Equal(t, "value1", v1)
	assert.Equal(t, int32(1), total)

	v2, err2 := rememberFunc("value2")
	assert.NoError(t, err2)
	assert.Equal(t, "value1", v2)
	assert.Equal(t, int32(1), total)

	time.Sleep(time.Millisecond * 200)
	v3, err3 := rememberFunc("value3")
	assert.NoError(t, err3)
	assert.Equal(t, "value3", v3)
	assert.Equal(t, int32(2), total)
}
