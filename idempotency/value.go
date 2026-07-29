package idempotency

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-fries/fries/codec/v4"
)

// ValueHandler is a business operation guarded by an [Executor] that returns a
// value.
type ValueHandler[T any] func(context.Context) (T, error)

// Result describes a value produced or replayed by [DoValue].
type Result[T any] struct {
	// Value is the value returned by the [ValueHandler] or decoded from a
	// completed record.
	Value T
	// Replayed reports whether Value came from a completed record.
	Replayed bool
}

// DoValue runs handler after atomically claiming key or returns the value
// stored by a previous successful execution.
//
// A completed key is decoded with the [Executor] codec and returned with
// [Result.Replayed] set to true. An active claim returns [ErrInProgress], and a
// fingerprint mismatch returns [ErrKeyConflict].
//
// Failures returned by handler abort the claim.
// Encoding failures leave the claim in progress until its execution TTL
// expires because the handler may already have produced business side effects.
// Failures from [Store.Complete] return the value produced by the handler
// together with the store error.
//
// A nil ctx returns [ErrInvalidContext], and an empty key returns
// [ErrInvalidKey]. DoValue panics if executor or handler is nil.
func DoValue[T any](
	ctx context.Context,
	executor *Executor,
	key string,
	handler ValueHandler[T],
	options ...ExecuteOption,
) (Result[T], error) {
	if executor == nil {
		panic("idempotency: nil executor")
	}
	if err := validateExecution(ctx, key); err != nil {
		return Result[T]{}, err
	}
	if handler == nil {
		panic("idempotency: nil value handler")
	}

	begin, err := executor.begin(ctx, key, options...)
	if err != nil {
		return Result[T]{}, err
	}
	if begin.replayed {
		result := Result[T]{Replayed: true}
		value, err := decodeValue[T](executor.config.codec, begin.result)
		if err != nil {
			return result, fmt.Errorf("idempotency: decode result: %w", err)
		}
		result.Value = value
		return result, nil
	}

	value, err := handler(ctx)
	result := Result[T]{Value: value}
	if err != nil {
		return result, errors.Join(err, begin.claim.abort(ctx))
	}

	encoded, err := executor.config.codec.Marshal(value)
	if err != nil {
		return result, fmt.Errorf("idempotency: encode result: %w", err)
	}
	if err := begin.claim.complete(ctx, encoded); err != nil {
		return result, err
	}
	return result, nil
}

func decodeValue[T any](valueCodec codec.Codec, data []byte) (T, error) {
	var value T
	valueType := reflect.TypeFor[T]()
	if valueType.Kind() != reflect.Pointer {
		return value, valueCodec.Unmarshal(data, &value)
	}

	destination := reflect.New(valueType.Elem())
	if err := valueCodec.Unmarshal(data, destination.Interface()); err != nil {
		return value, err
	}
	reflect.ValueOf(&value).Elem().Set(destination.Convert(valueType))
	return value, nil
}
