package cache

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func setPointerValue(dest, value any) error {
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
	synctest.Test(t, func(t *testing.T) {
		const ttl = time.Second
		ctx := t.Context()
		repo := NewRepository(newUtilMockStore())
		var calls int
		remember := func(loadValue string) string {
			t.Helper()
			value, err := Remember(ctx, repo, "test_key", ttl, func() (string, error) {
				calls++
				return loadValue, nil
			})
			require.NoError(t, err)
			return value
		}

		assert.Equal(t, "value1", remember("value1"))
		assert.Equal(t, 1, calls)

		time.Sleep(ttl - time.Nanosecond)
		assert.Equal(t, "value1", remember("before expiry"))
		assert.Equal(t, 1, calls)

		// The mock store keeps entries valid at their exact expiration time.
		time.Sleep(time.Nanosecond)
		assert.Equal(t, "value1", remember("at expiry"))
		assert.Equal(t, 1, calls)

		time.Sleep(time.Nanosecond)
		assert.Equal(t, "value2", remember("value2"))
		assert.Equal(t, 2, calls)

		time.Sleep(ttl - time.Nanosecond)
		assert.Equal(t, "value2", remember("cached again"))
		assert.Equal(t, 2, calls)
	})
}
