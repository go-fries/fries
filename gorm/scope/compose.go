package scope

import "gorm.io/gorm"

type Scopes []func(*gorm.DB) *gorm.DB
