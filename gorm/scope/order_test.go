package scope

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOrderBy(t *testing.T) {
	birthday1 := time.Now()
	birthday2 := time.Now()
	birthday3 := time.Now().Add(2 * time.Hour)
	birthday4 := time.Now().Add(3 * time.Hour)
	users := []*User{
		GetUser("OrderUser1", GetUserOptions{Age: 17, Birthday: &birthday1}),
		GetUser("OrderUser2", GetUserOptions{Age: 20, Birthday: &birthday2}),
		GetUser("OrderUser3", GetUserOptions{Age: 21, Birthday: &birthday3}),
		GetUser("OrderUser4", GetUserOptions{Age: 22, Birthday: &birthday4}),
	}

	CleanUsers()
	DB.Create(&users)

	// OrderBy
	var users1, users2, users3, users4 []User
	DB.Scopes(OrderBy("age").Scope()).Limit(2).Find(&users1)
	assert.Len(t, users1, 2)
	assert.Equal(t, "OrderUser1", users1[0].Name)
	assert.Equal(t, "OrderUser2", users1[1].Name)

	DB.Scopes(OrderBy("age", "asc").Scope()).Limit(2).Find(&users2)
	assert.Len(t, users2, 2)
	assert.Equal(t, "OrderUser1", users2[0].Name)
	assert.Equal(t, "OrderUser2", users2[1].Name)

	DB.Scopes(OrderBy("age", "desc").Scope()).Limit(2).Find(&users3)
	assert.Len(t, users3, 2)
	assert.Equal(t, "OrderUser4", users3[0].Name)
	assert.Equal(t, "OrderUser3", users3[1].Name)

	DB.Scopes(OrderBy("age", "unknown").Scope()).Limit(2).Find(&users4)
	assert.Len(t, users4, 2)
	assert.Equal(t, "OrderUser1", users4[0].Name)
	assert.Equal(t, "OrderUser2", users4[1].Name)

	// OrderByAsc
	var users5 []User
	DB.Scopes(OrderByAsc("age").Scope()).Limit(2).Find(&users5)
	assert.Len(t, users5, 2)
	assert.Equal(t, "OrderUser1", users5[0].Name)
	assert.Equal(t, "OrderUser2", users5[1].Name)

	// OrderByDesc
	var users6 []User
	DB.Scopes(OrderByDesc("age").Scope()).Limit(2).Find(&users6)
	assert.Len(t, users6, 2)
	assert.Equal(t, "OrderUser4", users6[0].Name)
	assert.Equal(t, "OrderUser3", users6[1].Name)

	// OrderByRaw
	var users7 []User
	DB.Scopes(OrderByRaw("age % 2 asc").Scope()).Limit(2).Find(&users7)
	assert.Len(t, users7, 2)
	assert.Equal(t, "OrderUser2", users7[0].Name)
	assert.Equal(t, "OrderUser4", users7[1].Name)

	// multiple OrderBy
	var users8 []User
	DB.Scopes(OrderBy("birthday", "asc").OrderBy("age", "desc").Scope()).Limit(2).Find(&users8)
	assert.Len(t, users8, 2)
	assert.Equal(t, "OrderUser2", users8[0].Name)
	assert.Equal(t, "OrderUser1", users8[1].Name)
}
