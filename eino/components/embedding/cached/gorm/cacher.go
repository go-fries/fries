package gorm

import (
	"context"
	"errors"
	"time"

	"github.com/go-fries/fries/eino/components/embedding/cached/v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Cacher struct {
	db *gorm.DB
}

var _ cached.Cacher = (*Cacher)(nil)

func NewCacher(db *gorm.DB) *Cacher {
	return &Cacher{
		db: db,
	}
}

func (c *Cacher) Set(ctx context.Context, key string, value []float64, expire time.Duration) error {
	if value == nil {
		return nil // No need to cache nil values
	}

	// Create or update the cache entry in the database
	cacheEntry := &Model{
		Key:     key,
		Vector:  datatypes.NewJSONType(value),
		Expired: time.Now().Add(expire),
	}

	return c.db.WithContext(ctx).Model(Model{}).Where("key = ?", key).
		Attrs(cacheEntry).
		FirstOrCreate(cacheEntry).Error
}

func (c *Cacher) Get(ctx context.Context, key string) ([]float64, error) {
	var cacheEntry Model
	if err := c.db.WithContext(ctx).Model(Model{}).
		Where("key = ?", key).
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
	return c.db.WithContext(ctx).Model(Model{}).Where("key = ?", key).Delete(&Model{}).Error
}

func (c *Cacher) CleanExpired(ctx context.Context) error {
	return c.db.WithContext(ctx).Model(Model{}).
		Where("expired < ?", time.Now()).
		Delete(&Model{}).Error
}
