package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerFunc_Handle(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("failed")
	wantTask := &Task{ID: "task-1"}
	var gotContext context.Context
	var gotTask *Task

	err := HandlerFunc(func(ctx context.Context, task *Task) error {
		gotContext = ctx
		gotTask = task
		return wantErr
	}).Handle(t.Context(), wantTask)

	require.ErrorIs(t, err, wantErr)
	assert.NotNil(t, gotContext)
	assert.Same(t, wantTask, gotTask)
}
