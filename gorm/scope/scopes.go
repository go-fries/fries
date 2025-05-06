package scope

import "gorm.io/gorm"

type Scopes []func(*gorm.DB) *gorm.DB

func (s Scopes) Apply(db *gorm.DB) *gorm.DB {
	return db.Scopes(s...)
}

func (s Scopes) Scope() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return s.Apply(db)
	}
}

func (s Scopes) Scopes() []func(*gorm.DB) *gorm.DB {
	return s
}

func (s Scopes) Add(scopes ...func(*gorm.DB) *gorm.DB) Scopes {
	return append(s, scopes...)
}
