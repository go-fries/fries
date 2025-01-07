package spt

import (
	"context"
	"testing"
	"time"

	"github.com/go-fries/fries/errors/v3"
	"github.com/stretchr/testify/assert"
)

type foo struct {
	Name string
	Age  int
}

func TestRetry(t *testing.T) {
	// success
	var i int
	err := Retry(func() error {
		i++
		if i < 3 {
			return assert.AnError
		}
		return nil
	}, 3)
	assert.Nil(t, err)
	assert.Equal(t, 3, i)

	// failed
	err = Retry(func() error {
		return assert.AnError
	}, 3)
	assert.Error(t, err)
	assert.Equal(t, 3, i)
}

func TestUntil(t *testing.T) {
	// no sleep
	var i int
	Until(func() bool {
		i++
		return i == 3
	})
	assert.Equal(t, 3, i)

	// has sleep
	i = 0
	now := time.Now()
	Until(func() bool {
		i++
		return i == 3
	}, 100*time.Millisecond)
	assert.Equal(t, 3, i)
	assert.True(t, time.Since(now) > 200*time.Millisecond)
}

func TestUntilTimeout(t *testing.T) {
	// success
	err := UntilTimeout(func() bool {
		time.Sleep(200 * time.Millisecond)
		return true
	}, 500*time.Millisecond)
	assert.Nil(t, err)

	// failed
	err = UntilTimeout(func() bool {
		time.Sleep(500 * time.Millisecond)
		return false
	}, 200*time.Millisecond)
	assert.Error(t, err)
	assert.True(t, errors.IsTimeoutError(err))
}

func TestTimeout(t *testing.T) {
	// success
	err := Timeout(func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}, 500*time.Millisecond)
	assert.Nil(t, err)

	// failed
	err = Timeout(func() error {
		time.Sleep(500 * time.Millisecond)
		return assert.AnError
	}, 200*time.Millisecond)
	assert.Error(t, err)
	assert.True(t, errors.IsTimeoutError(err))
}

func TestRepeat(t *testing.T) {
	// success
	var i int
	assert.NoError(t, Repeat(func() error { i++; return nil }, 3))
	assert.Equal(t, 3, i)

	// failed
	var j int
	assert.Error(t, Repeat(func() error {
		j++
		if j == 2 {
			return assert.AnError
		}
		return nil
	}, 3))
	assert.Equal(t, 2, j)
}

func TestPipe(t *testing.T) {
	// pipe functions
	pipe := Pipe(
		func(s string) string {
			return s + "1"
		},
		func(s string) string {
			return s + "2"
		},
		func(s string) string {
			return s + "3"
		},
	)

	assert.Equal(t, "0123", pipe("0"))

	// pipe functions
	pipe2 := Pipe(
		func(foo *foo) *foo {
			foo.Name = "bar" // nolint:goconst
			return foo
		},
		func(foo *foo) *foo {
			foo.Age = 18
			return foo
		},
	)

	f := &foo{Name: "foo"}
	assert.Equal(t, "foo", f.Name)
	assert.Equal(t, 0, f.Age)

	got := pipe2(f)
	assert.Equal(t, "bar", got.Name)
	assert.Equal(t, 18, got.Age)
}

func TestPipeWithErr(t *testing.T) {
	// pipe functions
	pipe := PipeWithErr(
		func(s string) (string, error) {
			return s + "1", nil
		},
		func(s string) (string, error) {
			return s + "2", nil
		},
		func(s string) (string, error) {
			return s + "3", nil
		},
	)

	got, err := pipe("0")
	assert.Nil(t, err)
	assert.Equal(t, "0123", got)

	// pipe functions
	pipe2 := PipeWithErr(
		func(foo *foo) (*foo, error) {
			foo.Name = "bar"
			return foo, nil
		},
		func(foo *foo) (*foo, error) {
			foo.Age = 18
			return foo, nil
		},
	)

	f := &foo{Name: "foo"}
	assert.Equal(t, "foo", f.Name)
	assert.Equal(t, 0, f.Age)

	got2, err := pipe2(f)
	assert.Nil(t, err)
	assert.Equal(t, "bar", got2.Name)
	assert.Equal(t, 18, got2.Age)

	// context
	pipe3 := PipeWithErr(
		func(ctx context.Context) (context.Context, error) {
			return context.WithValue(ctx, "foo", "bar"), nil //nolint:revive,staticcheck
		},
		func(ctx context.Context) (context.Context, error) {
			return context.WithValue(ctx, "bar", "baz"), nil //nolint:revive,staticcheck
		},
	)

	ctx, err := pipe3(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "bar", ctx.Value("foo"))
	assert.Equal(t, "baz", ctx.Value("bar"))

	// context with error
	pipe4 := PipeWithErr(
		func(ctx context.Context) (context.Context, error) {
			return context.WithValue(ctx, "foo", "bar"), nil //nolint:revive,staticcheck
		},
		func(context.Context) (context.Context, error) {
			return nil, assert.AnError
		},
	)

	ctx, err = pipe4(context.Background())
	assert.Error(t, err)
	assert.Nil(t, ctx)
}

func TestChain(t *testing.T) {
	chain := Chain(func(s string) string {
		return s + "1"
	}, func(s string) string {
		return s + "2"
	})

	got := chain("0")
	assert.Equal(t, "021", got)
}

func TestChainWithErr(t *testing.T) {
	chain := ChainWithErr(func(s string) (string, error) {
		return s + "1", nil
	}, func(s string) (string, error) {
		return s + "2", nil
	})

	got, err := chain("0")
	assert.Nil(t, err)
	assert.Equal(t, "021", got)

	// with error
	chain2 := ChainWithErr(func(s string) (string, error) {
		return s + "1", nil
	}, func(s string) (string, error) {
		return s + "2", assert.AnError
	})

	got, err = chain2("0")
	assert.Error(t, err)
	assert.Equal(t, "02", got)
}

func TestErrorIf(t *testing.T) {
	// if true
	got := ErrorIf(true, "foo")
	assert.Equal(t, "foo", got.Error())

	// if false
	got = ErrorIf(false, "foo")
	assert.Nil(t, got)

	// format
	got = ErrorIf(true, "foo %s", "bar")
	assert.Equal(t, "foo bar", got.Error())
}

func TestPanicIf(t *testing.T) {
	// if true
	assert.PanicsWithValue(t, "foo", func() {
		PanicIf(true, "foo")
	})

	// if false
	assert.NotPanics(t, func() {
		PanicIf(false, "foo")
	})

	// format
	assert.PanicsWithValue(t, "foo bar", func() {
		PanicIf(true, "foo %s", "bar")
	})

	// format
	assert.PanicsWithValue(t, "foo bar", func() {
		PanicIf(true, "foo %s", "bar")
	})
}

func TestScan_Basic(t *testing.T) {
	// string
	var foo string
	err := Scan("foo", &foo)
	assert.Nil(t, err)
	assert.Equal(t, "foo", foo)

	// int
	var bar int
	err = Scan(1, &bar)
	assert.Nil(t, err)
	assert.Equal(t, 1, bar)

	// struct
	type Baz struct {
		Name string
	}

	// struct.1
	var baz Baz
	err = Scan(Baz{
		Name: "baz",
	}, &baz)
	assert.Nil(t, err)
	assert.Equal(t, "baz", baz.Name)

	// struct.2
	assert.Error(t, Scan("foo", nil))
	var baz2 *Baz
	assert.Error(t, Scan("foo", baz2))

	// struct.3
	var baz3 Baz
	err = Scan(func() any {
		return Baz{
			Name: "baz",
		}
	}(), &baz3)
	assert.Nil(t, err)
	assert.Equal(t, "baz", baz3.Name)

	// test lower
	type test struct {
		Name string
	}
	var tt test
	err = Scan(func() any {
		return test{
			Name: "test",
		}
	}(), &tt)
	assert.Nil(t, err)
	assert.Equal(t, "test", tt.Name)
}

func TestScan_ComplexStruct(t *testing.T) {
	type AName struct {
		Name string
	}

	type ACompany struct {
		Name string
	}

	type A struct {
		Name      *AName
		Companies []*ACompany
	}

	type BName struct {
		Name string
	}

	type BCompany struct {
		Name string
	}

	type B struct {
		Name      *BName
		Companies []*BCompany
	}

	a := &A{
		Name: &AName{
			Name: "A",
		},
		Companies: []*ACompany{
			{
				Name: "A1",
			},
			{
				Name: "A2",
			},
		},
	}

	var b B
	err := Scan(a, &b)
	assert.Nil(t, err)
	assert.Equal(t, "A", b.Name.Name)
	assert.Equal(t, "A1", b.Companies[0].Name)
	assert.Equal(t, "A2", b.Companies[1].Name)
}

func TestOnce(t *testing.T) {
	var i int
	once := Once(func() {
		i++
	})
	once()
	once()
	assert.Equal(t, 1, i)
}

func BenchmarkOnce(b *testing.B) {
	var i int
	once := Once(func() {
		i++
	})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			once()
		}
	})

	assert.Equal(b, 1, i)
}

func TestWhen(t *testing.T) {
	// success
	var i int
	When(true, func() {
		i++
	})
	assert.Equal(t, 1, i)

	// failed
	When(false, func() {
		i++
	})
	assert.Equal(t, 1, i)
}
