package scope

import "gorm.io/gorm"

type Scopes []func(*gorm.DB) *gorm.DB

func New() *Scopes {
	return &Scopes{}
}

func (s *Scopes) Apply(db *gorm.DB) *gorm.DB {
	return db.Scopes(*s...)
}

func (s *Scopes) Scope() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return s.Apply(db)
	}
}

func (s *Scopes) Scopes() []func(*gorm.DB) *gorm.DB {
	return *s
}

func (s *Scopes) Add(scopes ...func(*gorm.DB) *gorm.DB) *Scopes {
	*s = append(*s, scopes...)
	return s
}
