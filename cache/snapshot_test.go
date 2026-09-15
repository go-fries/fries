package cache

import (
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type snapshotValue struct {
	value string
}

func TestSnapshot(t *testing.T) {
	var (
		snap  Snapshot[string, *snapshotValue]
		total atomic.Int32
	)

	for range 100 {
		value := snap.Lookup("key", func() *snapshotValue {
			total.Add(1)
			return &snapshotValue{value: "value"}
		})
		assert.Equal(t, "value", value.value)

		emptyValue := snap.Lookup("empty", func() *snapshotValue {
			total.Add(1)
			return nil
		})
		assert.Nil(t, emptyValue)
	}

	assert.Equal(t, int32(2), total.Load())
}

func TestSnapshot_Reset(t *testing.T) {
	var (
		snap  Snapshot[string, *snapshotValue]
		total atomic.Int32
	)

	for range 100 {
		value := snap.Lookup("key", func() *snapshotValue {
			total.Add(1)
			return &snapshotValue{value: "value"}
		})
		assert.Equal(t, "value", value.value)
	}

	assert.Equal(t, int32(1), total.Load())

	snap.Reset()

	for range 100 {
		value := snap.Lookup("key", func() *snapshotValue {
			total.Add(1)
			return &snapshotValue{value: "new_value"}
		})
		assert.Equal(t, "new_value", value.value)
	}

	assert.Equal(t, int32(2), total.Load())
}

func TestSnapshotWithErr(t *testing.T) {
	var (
		snap  SnapshotWithErr[string, *snapshotValue]
		total atomic.Int32
	)

	tests := []struct {
		name      string
		fn        func() (*snapshotValue, error)
		wantValue *snapshotValue
		wantErr   error
	}{
		{"with value, but no error", func() (*snapshotValue, error) {
			total.Add(1)
			return &snapshotValue{value: "value"}, nil
		}, &snapshotValue{value: "value"}, nil},
		{"with value and error", func() (*snapshotValue, error) {
			total.Add(1)
			return nil, assert.AnError
		}, nil, assert.AnError},
		{"empty value and nil error", func() (*snapshotValue, error) {
			total.Add(1)
			return nil, nil
		}, nil, nil},
		{"empty value and error", func() (*snapshotValue, error) {
			total.Add(1)
			return nil, assert.AnError
		}, nil, assert.AnError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 100 {
				value, err := snap.Lookup(tt.name, tt.fn)
				assert.Equal(t, tt.wantErr, err)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}

	assert.Equal(t, int32(len(tests)), total.Load())
}

func TestSnapshotWithErr_Reset(t *testing.T) {
	var (
		snap  SnapshotWithErr[string, *snapshotValue]
		total atomic.Int32
	)

	for range 100 {
		value, err := snap.Lookup("key", func() (*snapshotValue, error) {
			total.Add(1)
			return &snapshotValue{value: "value"}, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, "value", value.value)
	}

	assert.Equal(t, int32(1), total.Load())

	snap.Reset()

	for range 100 {
		value, err := snap.Lookup("key", func() (*snapshotValue, error) {
			total.Add(1)
			return &snapshotValue{value: "new_value"}, nil
		})
		assert.NoError(t, err)
		assert.Equal(t, "new_value", value.value)
	}

	assert.Equal(t, int32(2), total.Load())
}

func TestSnapshotWithExpireAndErr(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const ttl = time.Second
		var snap SnapshotWithExpireAndErr[string, *snapshotValue]
		var calls int
		lookup := func(loadValue string) *snapshotValue {
			t.Helper()
			value, err := snap.Lookup("key", func() (*snapshotValue, error) {
				calls++
				return &snapshotValue{value: loadValue}, nil
			}, ttl)
			require.NoError(t, err)
			return value
		}

		assert.Equal(t, &snapshotValue{value: "value"}, lookup("value"))
		assert.Equal(t, 1, calls)

		time.Sleep(ttl - time.Nanosecond)
		assert.Equal(t, &snapshotValue{value: "value"}, lookup("before expiry"))
		assert.Equal(t, 1, calls)

		// Expiration is exclusive: the cached value is still valid at the deadline.
		time.Sleep(time.Nanosecond)
		assert.Equal(t, &snapshotValue{value: "value"}, lookup("at expiry"))
		assert.Equal(t, 1, calls)

		time.Sleep(time.Nanosecond)
		assert.Equal(t, &snapshotValue{value: "new_value"}, lookup("new_value"))
		assert.Equal(t, 2, calls)

		time.Sleep(ttl - time.Nanosecond)
		assert.Equal(t, &snapshotValue{value: "new_value"}, lookup("cached again"))
		assert.Equal(t, 2, calls)
	})
}

func TestSnapshotWithExpireAndErr_Reset(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const ttl = time.Second
		var snap SnapshotWithExpireAndErr[string, *snapshotValue]
		var calls int
		lookup := func(loadValue string) *snapshotValue {
			t.Helper()
			value, err := snap.Lookup("key", func() (*snapshotValue, error) {
				calls++
				return &snapshotValue{value: loadValue}, nil
			}, ttl)
			require.NoError(t, err)
			return value
		}

		assert.Equal(t, &snapshotValue{value: "value"}, lookup("value"))
		assert.Equal(t, 1, calls)

		time.Sleep(ttl / 2)
		assert.Equal(t, &snapshotValue{value: "value"}, lookup("before reset"))
		assert.Equal(t, 1, calls)

		snap.Reset()
		assert.Equal(t, &snapshotValue{value: "new_value"}, lookup("new_value"))
		assert.Equal(t, 2, calls)

		// The replacement outlives the original entry's expiration time.
		time.Sleep(ttl/2 + time.Nanosecond)
		assert.Equal(t, &snapshotValue{value: "new_value"}, lookup("after reset"))
		assert.Equal(t, 2, calls)
	})
}
