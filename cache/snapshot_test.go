package cache

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type snapshotValue struct {
	value string
}

func TestSnapshot(t *testing.T) {
	var (
		snap  Snapshot[string, *snapshotValue]
		total atomic.Int32
	)

	for i := 0; i < 100; i++ {
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
			for i := 0; i < 100; i++ {
				value, err := snap.Lookup(tt.name, tt.fn)
				assert.Equal(t, tt.wantErr, err)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}

	assert.Equal(t, int32(len(tests)), total.Load())
}
