package errors

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	assert.True(t, IsBadRequest(BadRequest("BadRequest")))
	assert.True(t, IsUnauthorized(Unauthorized("Unauthorized")))
	assert.True(t, IsForbidden(Forbidden("Forbidden")))
	assert.True(t, IsNotFound(NotFound("NotFound")))
	assert.True(t, IsConflict(Conflict("Conflict")))
	assert.True(t, IsInternalServer(InternalServer("InternalServer")))
	assert.True(t, IsServiceUnavailable(ServiceUnavailable("ServiceUnavailable")))
	assert.True(t, IsGatewayTimeout(GatewayTimeout("GatewayTimeout")))
	assert.True(t, IsClientClosed(ClientClosed("ClientClosed")))
	assert.True(t, IsClientClosed(ClientClosed("ClientClosed")))
}

func TestErrors_New(t *testing.T) {
	assert.True(t, IsForbidden(New(http.StatusForbidden, "Forbidden")))
	assert.Equal(t, New(http.StatusForbidden, "Forbidden"), Forbidden("Forbidden"))
}

func TestErrors_Vars(t *testing.T) {
	assert.Equal(t, http.StatusText(http.StatusBadRequest), ErrBadRequest.Message)
	assert.Equal(t, http.StatusText(http.StatusUnauthorized), ErrUnauthorized.Message)
	assert.Equal(t, http.StatusText(http.StatusForbidden), ErrForbidden.Message)
	assert.Equal(t, http.StatusText(http.StatusNotFound), ErrNotFound.Message)
	assert.Equal(t, http.StatusText(http.StatusConflict), ErrConflict.Message)
	assert.Equal(t, http.StatusText(http.StatusInternalServerError), ErrInternalServer.Message)
	assert.Equal(t, http.StatusText(http.StatusServiceUnavailable), ErrServiceUnavailable.Message)
	assert.Equal(t, http.StatusText(http.StatusGatewayTimeout), ErrGatewayTimeout.Message)
	assert.Equal(t, "Client Closed", ErrClientClosed.Message)
}
