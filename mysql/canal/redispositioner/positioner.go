package redispositioner

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-fries/fries/codec/json/v3"
	"github.com/go-fries/fries/codec/v3"
	"github.com/go-fries/fries/mysql/canal/v3"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/redis/go-redis/v9"
)

const name = "canal:position"

var zeroPosition = mysql.Position{}

type Positioner struct {
	prefix string
	client redis.UniversalClient
	codec  codec.Codec
}

var _ canal.Positioner = (*Positioner)(nil)

type Option func(*Positioner)

func WithPrefix(prefix string) Option {
	return func(p *Positioner) {
		prefix = strings.TrimSuffix(prefix, ":")
		if prefix != "" {
			p.prefix = prefix + ":"
		}
	}
}

func WithCodec(codec codec.Codec) Option {
	return func(p *Positioner) {
		p.codec = codec
	}
}

func NewPositioner(client redis.UniversalClient, opts ...Option) *Positioner {
	positioner := &Positioner{
		client: client,
		codec:  json.Codec,
	}

	for _, opt := range opts {
		opt(positioner)
	}

	return positioner
}

func (p *Positioner) Get(ctx context.Context) (mysql.Position, error) {
	data, err := p.client.Get(ctx, p.prefix+name).Bytes()
	if err != nil {
		return zeroPosition, err
	}

	var pos mysql.Position
	if err := p.codec.Unmarshal(data, &pos); err != nil {
		return zeroPosition, err
	}

	return pos, nil
}

func (p *Positioner) Set(ctx context.Context, pos mysql.Position) error {
	data, err := p.codec.Marshal(pos)
	if err != nil {
		return err
	}

	if err := p.client.Set(ctx, p.prefix+name, data, redis.KeepTTL).Err(); err != nil {
		return err
	}

	return nil
}

// BufferedPositioner is a buffered version of Positioner. useful for high-frequency events.
type BufferedPositioner struct {
	*Positioner

	mu        sync.Mutex
	lastPos   mysql.Position // the last position in memory
	lastSaved mysql.Position // the last saved position in Redis

	// 缓冲策略配置
	flushInterval time.Duration // flush interval
	batchSize     int           // batch size
	timer         *time.Timer
	counter       int
}

var _ canal.Positioner = (*BufferedPositioner)(nil)

type BufferedOption func(*BufferedPositioner)

// WithFlushInterval is a buffered option to set the flush interval
func WithFlushInterval(interval time.Duration) BufferedOption {
	return func(p *BufferedPositioner) {
		p.flushInterval = interval
	}
}

// WithBatchSize is a buffered option to set the batch size
func WithBatchSize(size int) BufferedOption {
	return func(p *BufferedPositioner) {
		p.batchSize = size
	}
}

func NewBufferedPositioner(client redis.UniversalClient, opts ...Option) *BufferedPositioner {
	basePositioner := NewPositioner(client, opts...)

	buffered := &BufferedPositioner{
		Positioner:    basePositioner,
		flushInterval: 3 * time.Second,
		batchSize:     100,
	}

	buffered.startFlushTimer()

	return buffered
}

// ConfigureBuffered buffer parameters
func (p *BufferedPositioner) ConfigureBuffered(opts ...BufferedOption) {
	for _, opt := range opts {
		opt(p)
	}
}

// Start scheduled refresh task
func (p *BufferedPositioner) startFlushTimer() {
	p.timer = time.AfterFunc(p.flushInterval, func() {
		p.periodicFlush()
		p.timer.Reset(p.flushInterval)
	})
}

// Timed refresh function
func (p *BufferedPositioner) periodicFlush() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = p.Flush(ctx)
}

func (p *BufferedPositioner) Get(ctx context.Context) (mysql.Position, error) {
	p.mu.Lock()
	// if the last position in memory is valid, return it
	if p.isValidPosition(p.lastPos) {
		pos := p.lastPos
		p.mu.Unlock()
		return pos, nil
	}
	p.mu.Unlock()

	// else, get the position from Redis
	pos, err := p.Positioner.Get(ctx)
	if err != nil {
		return zeroPosition, err
	}

	// update the last position in memory
	p.mu.Lock()
	p.lastPos = pos
	p.lastSaved = pos
	p.mu.Unlock()

	return pos, nil
}

// 检查位置是否有效
func (p *BufferedPositioner) isValidPosition(pos mysql.Position) bool {
	return pos.Name != "" && pos.Pos != 0
}

// Check whether the location has undergone a substantive change
func (p *BufferedPositioner) hasSignificantChange(pos mysql.Position) bool {
	// A file is considered to have undergone a substantive change only if its name differs or its location changes beyond a certain threshold.
	// The logic here can be adjusted based on the actual situation
	return pos.Name != p.lastSaved.Name ||
		(pos.Pos > p.lastSaved.Pos && pos.Pos-p.lastSaved.Pos > 10000)
}

// Set rewrite the Set method using a buffering strategy
func (p *BufferedPositioner) Set(ctx context.Context, pos mysql.Position) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Update the location in memory
	p.lastPos = pos

	// counter increment
	p.counter++

	// Determine whether immediate writing to Redis is necessary:
	// 1. The file name has changed
	// 2. There has been a significant change in location
	// 3. The batch processing threshold has been reached
	if p.hasSignificantChange(pos) || p.counter >= p.batchSize {
		return p.doFlush(ctx)
	}

	return nil
}

// Flush refreshes the position to Redis immediately
func (p *BufferedPositioner) Flush(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.doFlush(ctx)
}

// doFlush 无锁的内部刷新函数
func (p *BufferedPositioner) doFlush(ctx context.Context) error {
	// if the last position in memory is the same as the last saved position, no need to update
	if p.lastPos.Name == p.lastSaved.Name && p.lastPos.Pos == p.lastSaved.Pos {
		return nil
	}

	// call the original Positioner Set method to actually write to Redis
	if err := p.Positioner.Set(ctx, p.lastPos); err != nil {
		return err
	}

	p.lastSaved = p.lastPos
	p.counter = 0

	return nil
}

// Close and clean up resources
func (p *BufferedPositioner) Close(ctx context.Context) {
	if p.timer != nil {
		p.timer.Stop()
	}

	_ = p.Flush(ctx)
}
