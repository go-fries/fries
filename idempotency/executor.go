package idempotency

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// Handler is an idempotent business operation.
type Handler func(context.Context) error

// Executor coordinates Handler execution through a Store.
type Executor struct {
	store  Store
	config config
}

// New creates an Executor.
//
// New panics if store is nil.
func New(store Store, options ...Option) *Executor {
	if store == nil {
		panic("idempotency: nil store")
	}
	return &Executor{
		store:  store,
		config: newConfig(options...),
	}
}

// Do runs handler after atomically claiming key.
//
// A completed key returns nil without running handler. An active claim returns
// ErrInProgress. A fingerprint mismatch returns ErrKeyConflict.
//
// Handler failures are followed by an attempt to abort the claim. Complete and
// Abort use a detached timeout context that preserves values from ctx. Do does
// not recover Handler panics; the claim remains until its execution TTL
// expires.
//
// A nil ctx returns ErrInvalidContext, and an empty key returns ErrInvalidKey.
// Do panics if handler is nil.
func (e *Executor) Do(
	ctx context.Context,
	key string,
	handler Handler,
	options ...ExecuteOption,
) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	if key == "" {
		return ErrInvalidKey
	}
	if handler == nil {
		panic("idempotency: nil handler")
	}
	if err := context.Cause(ctx); err != nil {
		return err
	}

	execution := newExecuteConfig(e.config, options...)
	token, err := newToken()
	if err != nil {
		return err
	}

	begin, err := e.store.Begin(ctx, BeginRequest{
		Key:         key,
		Token:       token,
		Fingerprint: execution.fingerprint,
		TTL:         execution.executionTTL,
	})
	if err != nil {
		return err
	}
	if err := context.Cause(ctx); err != nil {
		if begin.Status == BeginAcquired {
			return errors.Join(err, e.abort(ctx, key, token))
		}
		return err
	}

	switch begin.Status {
	case BeginInProgress:
		return ErrInProgress
	case BeginCompleted:
		return nil
	case BeginAcquired:
	default:
		return fmt.Errorf("idempotency: invalid begin status %d", begin.Status)
	}

	if err := handler(ctx); err != nil {
		return errors.Join(err, e.abort(ctx, key, token))
	}

	finalizationCtx, cancel := e.finalizationContext(ctx)
	defer cancel()
	return e.store.Complete(finalizationCtx, CompleteRequest{
		Key:   key,
		Token: token,
		TTL:   execution.resultTTL,
	})
}

func (e *Executor) abort(ctx context.Context, key, token string) error {
	finalizationCtx, cancel := e.finalizationContext(ctx)
	defer cancel()
	return e.store.Abort(finalizationCtx, AbortRequest{
		Key:   key,
		Token: token,
	})
}

func (e *Executor) finalizationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.WithoutCancel(ctx),
		e.config.finalizationTimeout,
	)
}

func newToken() (string, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", fmt.Errorf("idempotency: generate token: %w", err)
	}
	return hex.EncodeToString(token[:]), nil
}
