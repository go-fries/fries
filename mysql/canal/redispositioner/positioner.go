package redispositioner

import (
	"context"
	"strings"

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
