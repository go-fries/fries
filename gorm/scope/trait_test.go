package scope

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestTrait(t *testing.T) {
	users := []*User{
		GetUser("TraitWhenUser1", GetUserOptions{Age: 18}),
		GetUser("TraitWhenUser2", GetUserOptions{Age: 20}),
		GetUser("TraitWhenUser3", GetUserOptions{Age: 22}),
	}

	CleanUsers()
	DB.Create(&users)

	// When
	var users1, users2 []User
	DB.Scopes(When(true, func(db *gorm.DB) *gorm.DB {
		return db.Where("age > ?", 18)
	})).Find(&users1)
	assert.Len(t, users1, 2)
	assert.Equal(t, "TraitWhenUser2", users1[0].Name)
	assert.Equal(t, "TraitWhenUser3", users1[1].Name)

	DB.Scopes(When(false, func(db *gorm.DB) *gorm.DB {
		return db.Where("age > ?", 18)
	})).Find(&users2)
	assert.Len(t, users2, 3)

	// Unless
	var users3, users4 []User
	DB.Scopes(Unless(true, func(db *gorm.DB) *gorm.DB {
		return db.Where("age > ?", 18)
	})).Find(&users3)
	assert.Len(t, users3, 3)

	DB.Scopes(Unless(false, func(db *gorm.DB) *gorm.DB {
		return db.Where("age > ?", 18)
	})).Find(&users4)
	assert.Len(t, users4, 2)
	assert.Equal(t, "TraitWhenUser2", users4[0].Name)
	assert.Equal(t, "TraitWhenUser3", users4[1].Name)

	// multiple When and Unless
	var users5 []User
	DB.Scopes(When(true, func(db *gorm.DB) *gorm.DB {
		return db.Where("age > ?", 18)
	}), Unless(false, func(db *gorm.DB) *gorm.DB {
		return db.Where("age < ?", 22)
	})).Find(&users5)
	assert.Len(t, users5, 1)
	assert.Equal(t, "TraitWhenUser2", users5[0].Name)
}
