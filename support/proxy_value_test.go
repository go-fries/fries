package support

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type proxyMock struct {
	Name string
	Age  int
}

func TestProxy_Point(t *testing.T) {
	mock := &proxyMock{Name: "foo"}

	proxy := NewProxy(mock)
	assert.Equal(t, "foo", proxy.value.Name)
	assert.Equal(t, 0, proxy.value.Age)

	mock2 := proxy.Tap(func(p *proxyMock) {
		p.Name = "bar" //nolint:goconst
		p.Age = 18
	}).Value()
	assert.Equal(t, "bar", proxy.value.Name)
	assert.Equal(t, 18, proxy.value.Age)
	assert.Equal(t, "bar", mock.Name)
	assert.Equal(t, mock, mock2)

	mock3 := proxy.With(func(p *proxyMock) *proxyMock {
		p.Name = "baz" //nolint:goconst
		p.Age = 20
		return p
	}).Value()
	assert.Equal(t, "baz", proxy.value.Name)
	assert.Equal(t, 20, proxy.value.Age)
	assert.Equal(t, "baz", mock.Name)
	assert.Equal(t, mock, mock3)

	mock4 := proxy.When(true, func(p *proxyMock) *proxyMock {
		p.Name = "qux"
		p.Age = 22
		return p
	}).Value()
	assert.Equal(t, "qux", proxy.value.Name)
	assert.Equal(t, 22, proxy.value.Age)
	assert.Equal(t, "qux", mock.Name)
	assert.Equal(t, mock, mock4)

	mock5 := proxy.Value()
	assert.Equal(t, mock, mock5)
}

func TestProxy_Struct(t *testing.T) {
	mock := proxyMock{Name: "foo"}

	proxy := NewProxy(mock)
	assert.Equal(t, "foo", proxy.value.Name)
	assert.Equal(t, 0, proxy.value.Age)

	mock2 := proxy.Tap(func(p proxyMock) {
		p.Name = "bar"
		p.Age = 18
	}).Value()
	assert.Equal(t, "foo", proxy.value.Name)
	assert.Equal(t, 0, proxy.value.Age)
	assert.Equal(t, "foo", mock2.Name)

	mock3 := proxy.With(func(p proxyMock) proxyMock {
		p.Name = "baz"
		p.Age = 20
		return p
	}).Value()
	assert.Equal(t, "baz", proxy.value.Name)
	assert.Equal(t, 20, proxy.value.Age)
	assert.Equal(t, "baz", mock3.Name)
	assert.NotEqual(t, mock, mock3)

	mock4 := proxy.When(true, func(p proxyMock) proxyMock {
		p.Name = "qux"
		p.Age = 22
		return p
	}).Value()
	assert.Equal(t, "qux", proxy.value.Name)
	assert.Equal(t, 22, proxy.value.Age)
	assert.Equal(t, "qux", mock4.Name)
	assert.NotEqual(t, mock, mock4)

	mock5 := proxy.Value()
	assert.NotEqual(t, mock, mock5)

	mock6 := proxy.Unless(true, func(p proxyMock) proxyMock {
		p.Name = "quux"
		p.Age = 24
		return p
	}).Value()
	assert.Equal(t, "qux", proxy.value.Name)
	assert.Equal(t, 22, proxy.value.Age)
	assert.Equal(t, "qux", mock6.Name)
	assert.NotEqual(t, mock, mock6)
}
