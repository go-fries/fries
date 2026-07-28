package idempotency

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// Handler is a business operation guarded by an [Executor].
type Handler func(context.Context) error

// Executor coordinates [Handler] execution through a [Store].
type Executor struct {
	store  Store
	config config
}

// New creates an [Executor].
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
// [ErrInProgress]. A fingerprint mismatch returns [ErrKeyConflict].
//
// Failures returned by handler are followed by an attempt to abort the claim.
// [Store.Complete] and [Store.Abort] use a detached timeout context that
// preserves values from ctx. Do does not recover handler panics; the claim
// remains until its execution TTL expires.
//
// A nil ctx returns [ErrInvalidContext], and an empty key returns
// [ErrInvalidKey]. Do panics if handler is nil.
func (e *Executor) Do(
	ctx context.Context,
	key string,
	handler Handler,
	options ...ExecuteOption,
) error {
	if err := validateExecution(ctx, key); err != nil {
		return err
	}
	if handler == nil {
		panic("idempotency: nil handler")
	}

	begin, err := e.begin(ctx, key, options...)
	if err != nil {
		return err
	}
	if begin.replayed {
		return nil
	}

	if err := handler(ctx); err != nil {
		return errors.Join(err, begin.claim.abort(ctx))
	}
	return begin.claim.complete(ctx, nil)
}

type claim struct {
	executor  *Executor
	key       string
	token     string
	resultTTL time.Duration
}

type beginResult struct {
	claim    claim
	result   []byte
	replayed bool
}

func (e *Executor) begin(
	ctx context.Context,
	key string,
	options ...ExecuteOption,
) (beginResult, error) {
	if err := context.Cause(ctx); err != nil {
		return beginResult{}, err
	}

	execution := newExecuteConfig(e.config, options...)
	token, err := newToken()
	if err != nil {
		return beginResult{}, err
	}
	currentClaim := claim{
		executor:  e,
		key:       key,
		token:     token,
		resultTTL: execution.resultTTL,
	}

	result, err := e.store.Begin(ctx, BeginRequest{
		Key:         key,
		Token:       token,
		Fingerprint: execution.fingerprint,
		TTL:         execution.executionTTL,
	})
	if err != nil {
		return beginResult{}, err
	}
	if err := context.Cause(ctx); err != nil {
		if result.Status == BeginAcquired {
			return beginResult{}, errors.Join(err, currentClaim.abort(ctx))
		}
		return beginResult{}, err
	}

	switch result.Status {
	case BeginAcquired:
		return beginResult{claim: currentClaim}, nil
	case BeginInProgress:
		return beginResult{}, ErrInProgress
	case BeginCompleted:
		return beginResult{
			result:   result.Result,
			replayed: true,
		}, nil
	default:
		return beginResult{}, fmt.Errorf(
			"idempotency: invalid begin status %d",
			result.Status,
		)
	}
}

func (c claim) complete(ctx context.Context, result []byte) error {
	finalizationCtx, cancel := c.executor.finalizationContext(ctx)
	defer cancel()
	return c.executor.store.Complete(finalizationCtx, CompleteRequest{
		Key:    c.key,
		Token:  c.token,
		Result: result,
		TTL:    c.resultTTL,
	})
}

func (c claim) abort(ctx context.Context) error {
	finalizationCtx, cancel := c.executor.finalizationContext(ctx)
	defer cancel()
	return c.executor.store.Abort(finalizationCtx, AbortRequest{
		Key:   c.key,
		Token: c.token,
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

func validateExecution(ctx context.Context, key string) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	if key == "" {
		return ErrInvalidKey
	}
	return nil
}
