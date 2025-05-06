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

	var users1 []User

	DB.Scopes(scopes...).Find(&users1)
	assert.Len(t, users1, 2)
	assert.Equal(t, "ScopeUser1", users1[0].Name)
	assert.Equal(t, "ScopeUser2", users1[1].Name)
}
