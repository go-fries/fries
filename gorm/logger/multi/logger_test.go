package multi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

type recordingLogger struct {
	modeCalls []logger.LogLevel
	infos     []logCall
	warns     []logCall
	errors    []logCall
	traces    []traceCall
	filter    func(context.Context, string, ...any) (string, []any)
}

type logCall struct {
	ctx     context.Context
	message string
	data    []any
}

type traceCall struct {
	ctx          context.Context
	begin        time.Time
	sql          string
	rowsAffected int64
	err          error
}

var _ logger.Interface = (*recordingLogger)(nil)

func (r *recordingLogger) LogMode(level logger.LogLevel) logger.Interface {
	next := *r
	next.modeCalls = append(append([]logger.LogLevel{}, r.modeCalls...), level)
	next.infos = nil
	next.warns = nil
	next.errors = nil
	next.traces = nil
	return &next
}

func (r *recordingLogger) Info(ctx context.Context, msg string, data ...any) {
	r.infos = append(r.infos, logCall{ctx: ctx, message: msg, data: data})
}

func (r *recordingLogger) Warn(ctx context.Context, msg string, data ...any) {
	r.warns = append(r.warns, logCall{ctx: ctx, message: msg, data: data})
}

func (r *recordingLogger) Error(ctx context.Context, msg string, data ...any) {
	r.errors = append(r.errors, logCall{ctx: ctx, message: msg, data: data})
}

func (r *recordingLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rowsAffected := fc()
	r.traces = append(r.traces, traceCall{
		ctx:          ctx,
		begin:        begin,
		sql:          sql,
		rowsAffected: rowsAffected,
		err:          err,
	})
}

func (r *recordingLogger) ParamsFilter(ctx context.Context, sql string, params ...any) (string, []any) {
	if r.filter == nil {
		return sql, params
	}
	return r.filter(ctx, sql, params...)
}

func TestNewSkipsNilLoggers(t *testing.T) {
	l := New(nil)

	assert.Empty(t, l.loggers)
}

func TestLoggerDispatchesLogCalls(t *testing.T) {
	ctx := t.Context()
	first := &recordingLogger{}
	second := &recordingLogger{}
	l := New(first, second)

	l.Info(ctx, "info %s", "one")
	l.Warn(ctx, "warn %s", "two")
	l.Error(ctx, "error %s", "three")

	for _, item := range []*recordingLogger{first, second} {
		assert.Len(t, item.infos, 1)
		assert.Equal(t, ctx, item.infos[0].ctx)
		assert.Equal(t, "info %s", item.infos[0].message)
		assert.Equal(t, []any{"one"}, item.infos[0].data)

		assert.Len(t, item.warns, 1)
		assert.Equal(t, "warn %s", item.warns[0].message)
		assert.Equal(t, []any{"two"}, item.warns[0].data)

		assert.Len(t, item.errors, 1)
		assert.Equal(t, "error %s", item.errors[0].message)
		assert.Equal(t, []any{"three"}, item.errors[0].data)
	}
}

func TestLogModeAppliesToUnderlyingLoggers(t *testing.T) {
	first := &recordingLogger{}
	second := &recordingLogger{}
	l := New(first, second)

	leveled := l.LogMode(logger.Warn)
	leveled.Info(t.Context(), "message")

	for _, item := range []*recordingLogger{first, second} {
		assert.Empty(t, item.modeCalls)
		assert.Empty(t, item.infos)
	}

	leveledLogger, ok := leveled.(*Logger)
	assert.True(t, ok)
	assert.Len(t, leveledLogger.loggers, 2)

	for _, item := range leveledLogger.loggers {
		recorder, ok := item.(*recordingLogger)
		assert.True(t, ok)
		assert.Equal(t, []logger.LogLevel{logger.Warn}, recorder.modeCalls)
		assert.Len(t, recorder.infos, 1)
	}
}

func TestTraceDispatchesAndCachesTraceCallback(t *testing.T) {
	ctx := t.Context()
	begin := time.Now()
	err := errors.New("query failed")
	first := &recordingLogger{}
	second := &recordingLogger{}
	l := New(first, second)

	var calls int
	l.Trace(ctx, begin, func() (string, int64) {
		calls++
		return "select * from users", 3
	}, err)

	assert.Equal(t, 1, calls)
	for _, item := range []*recordingLogger{first, second} {
		assert.Len(t, item.traces, 1)
		assert.Equal(t, ctx, item.traces[0].ctx)
		assert.Equal(t, begin, item.traces[0].begin)
		assert.Equal(t, "select * from users", item.traces[0].sql)
		assert.Equal(t, int64(3), item.traces[0].rowsAffected)
		assert.Equal(t, err, item.traces[0].err)
	}
}

func TestTraceReturnsWithoutCallbackWhenNoLoggers(t *testing.T) {
	l := New()

	var calls int
	l.Trace(t.Context(), time.Now(), func() (string, int64) {
		calls++
		return "select * from users", 1
	}, nil)

	assert.Zero(t, calls)
}

func TestTraceDispatchesSingleLoggerWithoutExtraCallbackCalls(t *testing.T) {
	item := &recordingLogger{}
	l := New(item)

	var calls int
	l.Trace(t.Context(), time.Now(), func() (string, int64) {
		calls++
		return "select * from users", 1
	}, nil)

	assert.Equal(t, 1, calls)
	assert.Len(t, item.traces, 1)
	assert.Equal(t, "select * from users", item.traces[0].sql)
	assert.Equal(t, int64(1), item.traces[0].rowsAffected)
}

func TestTraceHandlesNilCallback(t *testing.T) {
	first := &recordingLogger{}
	second := &recordingLogger{}
	l := New(first, second)

	assert.NotPanics(t, func() {
		l.Trace(t.Context(), time.Now(), nil, nil)
	})

	for _, item := range []*recordingLogger{first, second} {
		assert.Len(t, item.traces, 1)
		assert.Empty(t, item.traces[0].sql)
		assert.Zero(t, item.traces[0].rowsAffected)
	}
}

func TestParamsFilterAppliesUnderlyingFiltersInOrder(t *testing.T) {
	ctx := t.Context()
	first := &recordingLogger{
		filter: func(got context.Context, sql string, params ...any) (string, []any) {
			assert.Equal(t, ctx, got)
			assert.Equal(t, "select * from users where name = ?", sql)
			assert.Equal(t, []any{"alice"}, params)
			return sql + " /* redacted */", []any{"***"}
		},
	}
	second := &recordingLogger{
		filter: func(got context.Context, sql string, params ...any) (string, []any) {
			assert.Equal(t, ctx, got)
			assert.Equal(t, "select * from users where name = ? /* redacted */", sql)
			assert.Equal(t, []any{"***"}, params)
			return sql, nil
		},
	}
	l := New(first, second)

	sql, params := l.ParamsFilter(ctx, "select * from users where name = ?", "alice")

	assert.Equal(t, "select * from users where name = ? /* redacted */", sql)
	assert.Nil(t, params)
}
