package sender

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	t.Parallel()

	client := &http.Client{Timeout: time.Second}
	validator := EndpointValidator(func(context.Context, *url.URL) error {
		return nil
	})
	c := newConfig(
		nil,
		WithHTTPClient(nil),
		WithTimeout(0),
		WithEndpointValidator(nil),
		WithHTTPClient(client),
		WithTimeout(time.Minute),
		WithEndpointValidator(validator),
		WithInsecureHTTP(),
	)

	assert.Same(t, client, c.client)
	assert.Equal(t, time.Minute, c.timeout)
	assert.True(t, c.allowHTTP)
	assert.Len(t, c.validators, 1)

	actual := c.newHTTPClient()
	assert.NotSame(t, client, actual)
	assert.Equal(t, time.Minute, actual.Timeout)
	assert.NotNil(t, actual.CheckRedirect)
	assert.Equal(
		t,
		http.ErrUseLastResponse,
		actual.CheckRedirect(nil, nil),
	)
	assert.Nil(t, client.CheckRedirect)

	preserved := newConfig(WithHTTPClient(client)).newHTTPClient()
	assert.Equal(t, time.Second, preserved.Timeout)

	defaulted := newConfig().newHTTPClient()
	assert.Equal(t, defaultTimeout, defaulted.Timeout)
}
