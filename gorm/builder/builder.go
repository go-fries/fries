package builder

import (
	"gorm.io/gorm"
)

type Builder[T any] struct {
	*gorm.DB
}

func New[T any](db *gorm.DB) *Builder[T] {
	return &Builder[T]{
		DB: db.Model(new(T)),
	}
}

// func (b *Builder[T]) Create(dest any) *gorm.DB {
// 	return b.DB.Create(dest)
// }

// func (b *Builder[T]) Count(count *int64) *gorm.DB {
// 	return b.DB.Count(count)
// }

// TODO：待验证，重复执行会不会有问题
func (b *Builder[T]) Page(page, pageSize int) *gorm.DB {
	return b.DB.Offset((page - 1) * pageSize).Limit(pageSize)
}

func (b *Builder[T]) Paginate(page, pageSize int) (dest []T, err error) {
	err = b.Page(page, pageSize).Find(&dest).Error
	return
}

// TODO: 待确认是否存在BUG
func (b *Builder[T]) Debug() *Builder[T] {
	return &Builder[T]{DB: b.DB.Debug()}
}
