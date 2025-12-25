package queue

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEmailPayload struct {
	To      string
	Subject string
}

type testSMSPayload struct {
	Phone   string
	Message string
}

func TestNewJobFor(t *testing.T) {
	payload := testEmailPayload{To: "test@example.com", Subject: "Hello"}
	job := NewJobFor(payload, WithQueue("emails"), WithPriority(10))

	assert.NotEmpty(t, job.ID())
	assert.Equal(t, "emails", job.Queue())
	assert.Equal(t, 10, job.Priority())
	assert.Equal(t, "queue.testEmailPayload", job.PayloadType())

	// Get typed payload
	p, err := job.Payload()
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", p.To)
	assert.Equal(t, "Hello", p.Subject)
}

func TestJobAs(t *testing.T) {
	// Create a job with NewJobFor
	original := NewJobFor(testEmailPayload{To: "test@example.com"})

	// Convert from Job to *JobFor[T]
	converted, err := JobAs[testEmailPayload](original.Job())
	require.NoError(t, err)

	p, err := converted.Payload()
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", p.To)
}

func TestJobAs_TypeMismatch(t *testing.T) {
	// Create an email job
	emailJob := NewJobFor(testEmailPayload{To: "test@example.com"})

	// Try to convert to SMS type
	_, err := JobAs[testSMSPayload](emailJob.Job())
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPayloadTypeMismatch)
}

func TestJobAs_LegacyJob(t *testing.T) {
	// Create a legacy job (without payloadType)
	legacyJob := NewJob(testEmailPayload{To: "test@example.com"})

	// Should still work via type assertion
	converted, err := JobAs[testEmailPayload](legacyJob)
	require.NoError(t, err)

	p, err := converted.Payload()
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", p.To)
}

func TestJobAs_LegacyJobTypeMismatch(t *testing.T) {
	// Create a legacy job
	legacyJob := NewJob(testEmailPayload{To: "test@example.com"})

	// Try to convert to wrong type
	_, err := JobAs[testSMSPayload](legacyJob)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPayloadTypeMismatch)
}

func TestMustJobAs(t *testing.T) {
	job := NewJobFor(testEmailPayload{To: "test@example.com"})

	// Should not panic
	converted := MustJobAs[testEmailPayload](job.Job())
	assert.Equal(t, "test@example.com", converted.MustPayload().To)
}

func TestMustJobAs_Panic(t *testing.T) {
	job := NewJobFor(testEmailPayload{To: "test@example.com"})

	// Should panic on type mismatch
	assert.Panics(t, func() {
		MustJobAs[testSMSPayload](job.Job())
	})
}

func TestHandlerFuncFor(t *testing.T) {
	var capturedPayload testEmailPayload
	var capturedJob Job

	handler := HandlerFuncFor[testEmailPayload](func(ctx context.Context, payload testEmailPayload, job Job) error {
		capturedPayload = payload
		capturedJob = job
		return nil
	})

	job := NewJobFor(testEmailPayload{To: "test@example.com", Subject: "Hello"})

	// Use as Handler interface
	err := handler.Handle(context.Background(), job.Job())
	require.NoError(t, err)

	assert.Equal(t, "test@example.com", capturedPayload.To)
	assert.Equal(t, "Hello", capturedPayload.Subject)
	assert.Equal(t, job.ID(), capturedJob.ID())
}

func TestHandlerFuncFor_TypeMismatch(t *testing.T) {
	handler := HandlerFuncFor[testSMSPayload](func(ctx context.Context, payload testSMSPayload, job Job) error {
		return nil
	})

	// Create an email job
	job := NewJobFor(testEmailPayload{To: "test@example.com"})

	// Should return error on type mismatch
	err := handler.Handle(context.Background(), job.Job())
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPayloadTypeMismatch)
}

func TestMultiHandler(t *testing.T) {
	var emailHandled, smsHandled bool

	multiHandler := NewMultiHandler(
		HandlerFuncFor[testEmailPayload](func(ctx context.Context, payload testEmailPayload, job Job) error {
			emailHandled = true
			return nil
		}),
		HandlerFuncFor[testSMSPayload](func(ctx context.Context, payload testSMSPayload, job Job) error {
			smsHandled = true
			return nil
		}),
	)

	// Handle email job
	emailJob := NewJobFor(testEmailPayload{To: "test@example.com"})
	err := multiHandler.Handle(context.Background(), emailJob.Job())
	require.NoError(t, err)
	assert.True(t, emailHandled)
	assert.False(t, smsHandled)

	// Reset
	emailHandled = false

	// Handle SMS job
	smsJob := NewJobFor(testSMSPayload{Phone: "123456"})
	err = multiHandler.Handle(context.Background(), smsJob.Job())
	require.NoError(t, err)
	assert.False(t, emailHandled)
	assert.True(t, smsHandled)
}

func TestMultiHandler_NoMatch(t *testing.T) {
	multiHandler := NewMultiHandler(
		HandlerFuncFor[testEmailPayload](func(ctx context.Context, payload testEmailPayload, job Job) error {
			return nil
		}),
	)

	// Create a job with different type
	smsJob := NewJobFor(testSMSPayload{Phone: "123456"})
	err := multiHandler.Handle(context.Background(), smsJob.Job())
	assert.Error(t, err)
}

func TestRegisterPayload(t *testing.T) {
	RegisterPayload[testEmailPayload]()
	RegisterPayload[testSMSPayload]()

	assert.True(t, isTypeRegistered("queue.testEmailPayload"))
	assert.True(t, isTypeRegistered("queue.testSMSPayload"))
	assert.False(t, isTypeRegistered("queue.unknownPayload"))
}
