package scope

import "gorm.io/gorm"

// When if condition is true, apply the scopes
//
//	When(true, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
//	When(false, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
func When(condition bool, f func(db *gorm.DB) *gorm.DB) Scopes {
	return Scopes{}.When(condition, f)
}

// Unless if condition is false, apply the scopes
//
//	Unless(false, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
//	Unless(true, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
func Unless(condition bool, f func(db *gorm.DB) *gorm.DB) Scopes {
	return Scopes{}.Unless(condition, f)
}

// When if condition is true, apply the scopes
//
//	When(true, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
//	When(false, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
func (s Scopes) When(condition bool, fc func(*gorm.DB) *gorm.DB) Scopes {
	if condition {
		return s.Add(fc)
	}
	return s
}

// Unless if condition is false, apply the scopes
//
//	Unless(false, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
//	Unless(true, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
func (s Scopes) Unless(condition bool, fc func(*gorm.DB) *gorm.DB) Scopes {
	return s.When(!condition, fc)
}
