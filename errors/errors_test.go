package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.True(t, IsTooManyRequests(TooManyRequests("TooManyRequests")))
	assert.True(t, IsPreconditionFailed(PreconditionFailed("PreconditionFailed")))
	assert.True(t, IsClientClosed(ClientClosed("ClientClosed")))
}

func TestErrors_New(t *testing.T) {
	assert.True(t, IsForbidden(New(http.StatusForbidden, "Forbidden")))
	assert.Equal(t, New(http.StatusForbidden, "Forbidden"), Forbidden("Forbidden"))
	assert.Equal(t, New(http.StatusTooManyRequests, "TooManyRequests"), TooManyRequests("TooManyRequests"))
	assert.Equal(t, New(http.StatusPreconditionFailed, "PreconditionFailed"), PreconditionFailed("PreconditionFailed"))
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
	assert.Equal(t, http.StatusText(http.StatusTooManyRequests), ErrTooManyRequests.Message)
	assert.Equal(t, http.StatusText(http.StatusPreconditionFailed), ErrPreconditionFailed.Message)
	assert.Equal(t, "Client Closed", ErrClientClosed.Message)
}

func TestErrors_WrappedStatusHelpers(t *testing.T) {
	assert.True(t, IsTooManyRequests(fmt.Errorf("wrapped: %w", TooManyRequests("rate limited"))))
	assert.True(t, IsPreconditionFailed(fmt.Errorf("wrapped: %w", PreconditionFailed("etag mismatch"))))
}

func TestErrors_StandardLibraryWrappers(t *testing.T) {
	sentinel := stderrors.New("sentinel")
	wrapped := fmt.Errorf("wrapped: %w", sentinel)

	assert.True(t, Is(wrapped, sentinel))
	assert.Same(t, sentinel, Unwrap(wrapped))

	joined := Join(sentinel, ErrUnsupported)
	assert.True(t, Is(joined, sentinel))
	assert.True(t, Is(joined, ErrUnsupported))

	err := fmt.Errorf("wrapped: %w", BadRequest("bad request"))
	var target *kratoserrors.Error
	require.True(t, As(err, &target))
	assert.Equal(t, int32(http.StatusBadRequest), target.Code)
}
