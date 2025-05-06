package scope

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestScopes(t *testing.T) {
	users := []*User{
		GetUser("ScopeUser1", GetUserOptions{}),
		GetUser("ScopeUser2", GetUserOptions{}),
		GetUser("ScopeUser3", GetUserOptions{}),
	}

	scopes := Scopes{}.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where("name in (?)", []string{"ScopeUser1", "ScopeUser2"})
	})

	CleanUsers()
	DB.Create(&users)

	var users1, users2, users3, users4 []User

	// Scope/Add
	DB.Scopes(scopes.Scope()).Find(&users1)
	assert.Len(t, users1, 2)
	assert.Equal(t, "ScopeUser1", users1[0].Name)
	assert.Equal(t, "ScopeUser2", users1[1].Name)

	// Apply
	scopes.Apply(DB).Find(&users2)
	assert.Len(t, users2, 2)

	DB.Scopes(scopes.Apply).Find(&users3)
	assert.Len(t, users3, 2)

	// Scopes
	DB.Scopes(scopes.Scopes()...).Find(&users4)
	assert.Len(t, users4, 2)
}
