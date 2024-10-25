package jet

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransporter_HTTPTransporter(t *testing.T) {
	testRequest := []byte("test-request")
	testResponse := []byte("test-response")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r.Body)

		assert.Equal(t, testRequest, buf.Bytes())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testResponse)
	}))
	defer srv.Close()

	// create a new http transporter
	transport, err := NewHTTPTransporter(WithHTTPTransporterAddr(srv.URL))
	assert.NoError(t, err)

	// send a request
	response, err := transport.Send(context.Background(), testRequest)
	assert.NoError(t, err)
	assert.Equal(t, testResponse, response)
}

func TestTransporter_HTTPTransporter_InvalidErrs(t *testing.T) {
	_, err := NewHTTPTransporter()
	assert.Error(t, err)
	assert.Equal(t, ErrHTTPTransporterAddrIsRequired, err)

	_, err = NewHTTPTransporter(WithHTTPTransporterAddr("test"), WithHTTPTransporterClient(nil))
	assert.Error(t, err)
	assert.Equal(t, ErrorHTTPTransporterClientIsRequired, err)
}

func TestTransporter_HTTPTransporter_HTTPTransporterServerError(t *testing.T) {
	err := &HTTPTransporterServerError{
		StatusCode: 500,
		Err:        errors.New("custom error"),
	}
	assert.True(t, IsHTTPTransporterServerError(err))
	assert.IsType(t, "", err.Error())
	assert.Equal(t, "custom error", err.Unwrap().Error())
}
