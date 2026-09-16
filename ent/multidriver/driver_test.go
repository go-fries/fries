package multidriver

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type closeDriver struct {
	dialect.Driver
	closed atomic.Int64
	close  func() error
}

func (d *closeDriver) Close() error {
	d.closed.Add(1)
	if d.close != nil {
		return d.close()
	}
	return nil
}

func (d *closeDriver) Query(context.Context, string, any, any) error { return nil }

func TestCloseDefaultReader(t *testing.T) {
	t.Parallel()
	writer := &closeDriver{}
	d, err := New(WithWriter(writer))
	require.NoError(t, err)
	require.NoError(t, d.Close())
	assert.EqualValues(t, 1, writer.closed.Load())
	// Repeated calls must not close the underlying resource again.
	require.NoError(t, d.Close())
	assert.EqualValues(t, 1, writer.closed.Load())
}

func TestCloseSharedDrivers(t *testing.T) {
	t.Parallel()
	writer, reader, other := &closeDriver{}, &closeDriver{}, &closeDriver{}
	d, err := New(WithWriter(writer), WithReaders(writer, reader, reader, other, writer))
	require.NoError(t, err)
	require.NoError(t, d.Close())
	for _, driver := range []*closeDriver{writer, reader, other} {
		assert.EqualValues(t, 1, driver.closed.Load())
	}
}

func TestCloseComparableValues(t *testing.T) {
	t.Parallel()
	writer := &closeDriver{}
	value := interfaceDriver{closeDriver: writer, data: "same driver"}
	d, err := New(WithWriter(value), WithReaders(value, value))
	require.NoError(t, err)
	require.NoError(t, d.Close())
	assert.EqualValues(t, 1, writer.closed.Load())
}

type closeFailure struct{ name string }

func (e *closeFailure) Error() string { return e.name }

func TestClosePreservesErrorsAndOrder(t *testing.T) {
	t.Parallel()
	writerErr := &closeFailure{name: "writer failed"}
	readerErr := errors.New("reader failed")
	var order []string
	writer := &closeDriver{close: func() error { order = append(order, "writer"); return writerErr }}
	reader := &closeDriver{close: func() error { order = append(order, "reader"); return readerErr }}
	other := &closeDriver{close: func() error { order = append(order, "other"); return nil }}
	d, err := New(WithWriter(writer), WithReaders(reader, reader, writer, other))
	require.NoError(t, err)
	err = d.Close()
	require.ErrorIs(t, err, ErrClose)
	assert.ErrorIs(t, err, writerErr)
	assert.ErrorIs(t, err, readerErr)
	var failure *closeFailure
	if assert.ErrorAs(t, err, &failure) {
		assert.Same(t, writerErr, failure)
	}
	assert.Contains(t, err.Error(), "writer failed")
	assert.Contains(t, err.Error(), "reader failed")
	assert.Equal(t, []string{"writer", "reader", "other"}, order)
	assert.Same(t, err, d.Close())
	assert.Equal(t, []string{"writer", "reader", "other"}, order)
}

func TestCloseConcurrent(t *testing.T) {
	t.Parallel()
	cause := errors.New("close failed")
	writer := &closeDriver{close: func() error { return cause }}
	reader := &closeDriver{}
	d, err := New(WithWriter(writer), WithReaders(reader, writer, reader))
	require.NoError(t, err)
	const callers = 32
	start := make(chan struct{})
	results := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			<-start
			results <- d.Close()
		})
	}
	close(start)
	wg.Wait()
	close(results)
	first := <-results
	require.ErrorIs(t, first, ErrClose)
	assert.ErrorIs(t, first, cause)
	for err := range results {
		assert.Same(t, first, err)
	}
	assert.EqualValues(t, 1, writer.closed.Load())
	assert.EqualValues(t, 1, reader.closed.Load())
}

// A legal dialect.Driver value with a slice cannot be compared or used as a map key.
type valueDriver struct {
	*closeDriver
	data []byte
}

// The type is comparable, but a contained value can still make equality panic.
type interfaceDriver struct {
	*closeDriver
	data any
}

func TestCloseNonComparableDrivers(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		wrap func(*closeDriver) dialect.Driver
	}{
		{"slice", func(d *closeDriver) dialect.Driver { return valueDriver{closeDriver: d, data: []byte{1}} }},
		{"interface with slice", func(d *closeDriver) dialect.Driver {
			return interfaceDriver{closeDriver: d, data: []byte{1}}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			writer := &closeDriver{}
			d, err := New(WithWriter(tt.wrap(writer)))
			require.NoError(t, err)
			require.NoError(t, d.Close())
			assert.EqualValues(t, 1, writer.closed.Load())

			reader := &closeDriver{}
			value := tt.wrap(reader)
			d, err = New(WithWriter(&closeDriver{}), WithReaders(value, value))
			require.NoError(t, err)
			require.NoError(t, d.Close())
			assert.EqualValues(t, 2, reader.closed.Load())
		})
	}
}

func TestReaderSnapshotPreservesWeights(t *testing.T) {
	t.Parallel()
	writer, reader, other := &closeDriver{}, &closeDriver{}, &closeDriver{}
	readers := []dialect.Driver{reader, reader, writer}
	want := []dialect.Driver{reader, reader, writer}
	policy := PolicyFunc(func(got []dialect.Driver) dialect.Driver {
		require.Len(t, got, len(want))
		for i := range want {
			assert.Same(t, want[i], got[i])
		}
		return got[0]
	})
	d, err := New(WithWriter(writer), WithReaders(readers...), WithPolicy(policy))
	require.NoError(t, err)
	readers[0], readers[1] = other, other
	ctx := ent.NewQueryContext(t.Context(), &ent.QueryContext{})
	require.NoError(t, d.Query(ctx, "SELECT 1", nil, nil))
	require.NoError(t, d.Close())
	assert.EqualValues(t, 1, writer.closed.Load())
	assert.EqualValues(t, 1, reader.closed.Load())
	assert.Zero(t, other.closed.Load())
}
