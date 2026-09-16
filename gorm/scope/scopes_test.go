package scope

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestScopes(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("ScopeUser1", GetUserOptions{}),
		GetUser("ScopeUser2", GetUserOptions{}),
		GetUser("ScopeUser3", GetUserOptions{}),
	}

	scopes := Scopes{}.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where("name in (?)", []string{"ScopeUser1", "ScopeUser2"})
	})

	require.NoError(t, db.Create(&users).Error)

	var users1 []User

	require.NoError(t, db.Order("id").Scopes(scopes...).Find(&users1).Error)
	require.Len(t, users1, 2)
	assert.Equal(t, "ScopeUser1", users1[0].Name)
	assert.Equal(t, "ScopeUser2", users1[1].Name)
}
