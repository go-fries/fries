package scope

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPagination(t *testing.T) {
	users := []*User{
		GetUser("PaginationUser1", GetUserOptions{}),
		GetUser("PaginationUser2", GetUserOptions{}),
		GetUser("PaginationUser3", GetUserOptions{}),
		GetUser("PaginationUser4", GetUserOptions{}),
		GetUser("PaginationUser5", GetUserOptions{}),
	}

	CleanUsers()
	DB.Create(&users)

	var users1, users2, users3, users4, users5, users6 []User

	// Offset&Limit
	DB.Scopes(Offset(2).Limit(2).Scope()).Find(&users1)
	assert.Len(t, users1, 2)
	assert.Equal(t, "PaginationUser3", users1[0].Name)
	assert.Equal(t, "PaginationUser4", users1[1].Name)

	DB.Scopes(Limit(2).Skip(4).Scope()).Find(&users2)
	assert.Len(t, users2, 1)
	assert.Equal(t, "PaginationUser5", users2[0].Name)

	// Skip&Take
	DB.Scopes(Skip(2).Take(2).Scope()).Find(&users3)
	assert.Len(t, users3, 2)
	assert.Equal(t, "PaginationUser3", users3[0].Name)
	assert.Equal(t, "PaginationUser4", users3[1].Name)

	DB.Scopes(Take(2).Skip(4).Scope()).Find(&users4)
	assert.Len(t, users4, 1)
	assert.Equal(t, "PaginationUser5", users4[0].Name)

	// Page
	DB.Scopes(Page(2, 2).Scope()).Find(&users5)
	assert.Len(t, users5, 2)
	assert.Equal(t, "PaginationUser3", users5[0].Name)

	DB.Scopes(Page(3, 2).Scope()).Find(&users6)
	assert.Len(t, users6, 1)
	assert.Equal(t, "PaginationUser5", users6[0].Name)
}
