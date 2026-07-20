package response_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-fries/fries/http/response/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	body := response.Success(
		"working properly",
		map[string]int{"id": 11},
		response.WithCode(10000),
	)

	err := response.Write(recorder, http.StatusCreated, body)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.JSONEq(t, `{
		"status": true,
		"code": 10000,
		"message": "working properly",
		"data": {"id": 11}
	}`, recorder.Body.String())
	assert.Equal(t, byte('\n'), recorder.Body.Bytes()[recorder.Body.Len()-1])
}

func TestWriteEncodingErrorDoesNotCommitResponse(t *testing.T) {
	t.Parallel()

	writer := newRecordingWriter()
	body := response.Success("ok", make(chan int))

	err := response.Write(writer, http.StatusOK, body)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "response: marshal body")
	assert.False(t, writer.committed)
	assert.Empty(t, writer.header)
	assert.Empty(t, writer.body)
}

func TestWriteReturnsWriterError(t *testing.T) {
	t.Parallel()

	writeErr := errors.New("write failed")
	writer := newRecordingWriter()
	writer.writeErr = writeErr

	err := response.Write(writer, http.StatusAccepted, response.Success("ok", nil))

	require.ErrorIs(t, err, writeErr)
	assert.True(t, writer.committed)
	assert.Equal(t, http.StatusAccepted, writer.statusCode)
}

func TestWritePanicsWithNilWriter(t *testing.T) {
	t.Parallel()

	assert.PanicsWithValue(t, "response: nil response writer", func() {
		_ = response.Write(nil, http.StatusOK, response.Success("ok", nil))
	})
}

type recordingWriter struct {
	header     http.Header
	body       []byte
	statusCode int
	committed  bool
	writeErr   error
}

func newRecordingWriter() *recordingWriter {
	return &recordingWriter{header: make(http.Header)}
}

func (w *recordingWriter) Header() http.Header {
	return w.header
}

func (w *recordingWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.committed = true
}

func (w *recordingWriter) Write(payload []byte) (int, error) {
	w.body = append(w.body, payload...)
	return len(payload), w.writeErr
}
