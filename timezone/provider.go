package timezone

import (
	"context"
	"time"
)

var (
	UTC       = time.UTC
	PRC, _    = time.LoadLocation("Asia/Shanghai")
	Taipei, _ = time.LoadLocation("Asia/Taipei")
)

func MustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type Provider struct {
	local *time.Location
}

func NewProvider(local *time.Location) *Provider {
	return &Provider{local: local}
}

func (p *Provider) Bootstrap(ctx context.Context) (context.Context, error) {
	time.Local = p.local
	return NewContext(ctx, p.local), nil
}

func (p *Provider) Shutdown(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
