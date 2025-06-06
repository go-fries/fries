package gorm

import (
	"context"
	"errors"
	"time"

	"github.com/go-fries/fries/eino/components/embedding/cached/v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Cacher struct {
	db    *gorm.DB
	model any // model type for the cache entries
}

var _ cached.Cacher = (*Cacher)(nil)

type Option func(*Cacher)

func WithModel(model any) Option {
	return func(c *Cacher) {
		c.model = model
	}
}

func NewCacher(db *gorm.DB, opts ...Option) *Cacher {
	cacher := &Cacher{
		db:    db,
		model: &Model{}, // Default Model type
	}
	for _, opt := range opts {
		opt(cacher)
	}
	return cacher
}

func (c *Cacher) Set(ctx context.Context, key string, value []float64, expire time.Duration) error {
	return c.db.WithContext(ctx).Model(c.model).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"vector", "expired"}),
		}).
		Create(map[string]any{
			"key":     key,
			"vector":  datatypes.NewJSONType(value),
			"expired": time.Now().Add(expire),
		}).Error
}

func (c *Cacher) Get(ctx context.Context, key string) ([]float64, error) {
	var cacheEntry Model
	if err := c.db.WithContext(ctx).Model(c.model).
		Where("`key` = ?", key).
		First(&cacheEntry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, cached.ErrCacherKeyNotFound
		}
		return nil, err // Other errors
	}

	if time.Now().After(cacheEntry.Expired) {
		_ = c.Delete(ctx, key)                  // Clean up expired cache entry
		return nil, cached.ErrCacherKeyNotFound // Return error if the cache entry is expired
	}

	return cacheEntry.Vector.Data(), nil
}

func (c *Cacher) Delete(ctx context.Context, key string) error {
	return c.db.WithContext(ctx).Model(c.model).Where("`key` = ?", key).Delete(c.model).Error
}

// func (c *Cacher) CleanExpired(ctx context.Context) error {
// 	return c.db.WithContext(ctx).Model(c.model).
// 		Where("expired < ?", time.Now()).
// 		Delete(c.model).Error
// }
