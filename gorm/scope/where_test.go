package scope

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhere_Where(t *testing.T) {
	users := []*User{
		GetUser("WhereUser1", GetUserOptions{}),
		GetUser("WhereUser2", GetUserOptions{}),
		GetUser("WhereUser3", GetUserOptions{}),
	}

	CleanUsers()
	DB.Create(&users)

	var users1, users2, users3 []User
	DB.Scopes(Where("name in (?)", []string{"WhereUser1", "WhereUser2"}).Scope()).Find(&users1)
	assert.Len(t, users1, 2)

	DB.Scopes(Where("name in (?)", []string{"WhereUser1", "WhereUser4"}).Scope()).Find(&users2)
	assert.Len(t, users2, 1)

	DB.Scopes(Where("name = ?", "WhereUser3").Scope()).Find(&users3)
	assert.Len(t, users3, 1)
	assert.Equal(t, "WhereUser3", users3[0].Name)
}

func TestWhere_Between(t *testing.T) {
	users := []*User{
		GetUser("WhereBetweenUser1", GetUserOptions{Age: 18}),
		GetUser("WhereBetweenUser2", GetUserOptions{Age: 20}),
		GetUser("WhereBetweenUser3", GetUserOptions{Age: 22}),
	}

	CleanUsers()
	DB.Create(&users)

	var users1, users2, users3 []User
	DB.Scopes(WhereBetween("age", 18, 20).Scope()).Find(&users1)
	assert.Len(t, users1, 2)

	DB.Scopes(WhereBetween("age", 18, 19).Scope()).Find(&users2)
	assert.Len(t, users2, 1)
	assert.Equal(t, "WhereBetweenUser1", users2[0].Name)

	DB.Scopes(WhereBetween("age", 12, 16).Scope()).Find(&users3)
	assert.Len(t, users3, 0)

	var users4, users5, users6 []User
	DB.Scopes(WhereNotBetween("age", 18, 20).Scope()).Find(&users4)
	assert.Len(t, users4, 1)
	assert.Equal(t, "WhereBetweenUser3", users4[0].Name)

	DB.Scopes(WhereNotBetween("age", 18, 19).Scope()).Find(&users5)
	assert.Len(t, users5, 2)

	DB.Scopes(WhereNotBetween("age", 12, 16).Scope()).Find(&users6)
	assert.Len(t, users6, 3)
}

func TestWhere_In(t *testing.T) {
	users := []*User{
		GetUser("WhereInUser1", GetUserOptions{Age: 18}),
		GetUser("WhereInUser2", GetUserOptions{Age: 20}),
		GetUser("WhereInUser3", GetUserOptions{Age: 22}),
	}

	CleanUsers()
	DB.Create(&users)

	var users1, users2, users3 []User
	DB.Debug().Scopes(WhereIn("name", "WhereInUser1", "WhereInUser2").Scope()).Find(&users1)
	assert.Len(t, users1, 2)

	DB.Scopes(WhereIn("age", []int{18, 20}).Scope()).Find(&users2)
	assert.Len(t, users2, 2)

	DB.Scopes(WhereIn("name", []string{"WhereInUser1", "WhereInUser2"}).Scope()).Find(&users3)
	assert.Len(t, users3, 2)

	var users4, users5, users6 []User
	DB.Scopes(WhereNotIn("name", "WhereInUser1", "WhereInUser2").Scope()).Find(&users4)
	assert.Len(t, users4, 1)
	assert.Equal(t, "WhereInUser3", users4[0].Name)

	DB.Scopes(WhereNotIn("age", []int{18, 20}).Scope()).Find(&users5)
	assert.Len(t, users5, 1)
	assert.Equal(t, "WhereInUser3", users5[0].Name)

	DB.Scopes(WhereNotIn("name", []string{"WhereInUser1", "WhereInUser2"}).Scope()).Find(&users6)
	assert.Len(t, users6, 1)
	assert.Equal(t, "WhereInUser3", users6[0].Name)
}

func TestWhere_Like(t *testing.T) {
	users := []*User{
		GetUser("WhereLikeUser1", GetUserOptions{Age: 18}),
		GetUser("WhereLikeUser2", GetUserOptions{Age: 20}),
		GetUser("WhereLikeUser3", GetUserOptions{Age: 22}),
	}

	CleanUsers()
	DB.Create(&users)

	var users1, users2, users3, users4 []User
	DB.Scopes(WhereLike("name", "WhereLikeUser1").Scope()).Find(&users1)
	assert.Len(t, users1, 1)
	assert.Equal(t, "WhereLikeUser1", users1[0].Name)

	DB.Scopes(WhereLike("name", "WhereLike%").Scope()).Find(&users2)
	assert.Len(t, users2, 3)

	DB.Scopes(WhereLike("name", "%LikeUser3").Scope()).Find(&users3)
	assert.Len(t, users3, 1)
	assert.Equal(t, "WhereLikeUser3", users3[0].Name)

	DB.Scopes(WhereLike("name", "%Like%").Scope()).Find(&users4)
	assert.Len(t, users4, 3)

	var users5, users6, users7, users8 []User
	DB.Scopes(WhereNotLike("name", "WhereLikeUser1").Scope()).Find(&users5)
	assert.Len(t, users5, 2)

	DB.Scopes(WhereNotLike("name", "WhereLike%").Scope()).Find(&users6)
	assert.Len(t, users6, 0)

	DB.Scopes(WhereNotLike("name", "%LikeUser3").Scope()).Find(&users7)
	assert.Len(t, users7, 2)

	DB.Scopes(WhereNotLike("name", "%Like%").Scope()).Find(&users8)
	assert.Len(t, users8, 0)
}

func TestWhere_OP(t *testing.T) {
	users := []*User{
		GetUser("WhereLikeUser1", GetUserOptions{Age: 18}),
		GetUser("WhereLikeUser2", GetUserOptions{Age: 20}),
		GetUser("WhereLikeUser3", GetUserOptions{Age: 22}),
		GetUser("WhereLikeUser4", GetUserOptions{Age: 22}),
	}

	CleanUsers()
	DB.Create(&users)

	// Eq
	var users1, users2, users3 []User
	DB.Scopes(WhereEq("name", "WhereLikeUser1").Scope()).Find(&users1)
	assert.Len(t, users1, 1)
	assert.Equal(t, "WhereLikeUser1", users1[0].Name)

	DB.Scopes(WhereEq("age", 18).Scope()).Find(&users2)
	assert.Len(t, users2, 1)
	assert.Equal(t, "WhereLikeUser1", users2[0].Name)

	DB.Scopes(WhereEq("age", 22).Scope()).Find(&users3)
	assert.Len(t, users3, 2)

	// Egt
	var users4, users5, users6 []User
	DB.Scopes(WhereEgt("age", 20).Scope()).Find(&users4)
	assert.Len(t, users4, 3)

	DB.Scopes(WhereEgt("age", 22).Scope()).Find(&users5)
	assert.Len(t, users5, 2)

	DB.Scopes(WhereEgt("age", 23).Scope()).Find(&users6)
	assert.Len(t, users6, 0)

	// Elt
	var users7, users8, users9 []User
	DB.Scopes(WhereElt("age", 20).Scope()).Find(&users7)
	assert.Len(t, users7, 2)

	DB.Scopes(WhereElt("age", 22).Scope()).Find(&users8)
	assert.Len(t, users8, 4)

	DB.Scopes(WhereElt("age", 18).Scope()).Find(&users9)
	assert.Len(t, users9, 1)

	// Gt
	var users10, users11, users12 []User
	DB.Scopes(WhereGt("age", 20).Scope()).Find(&users10)
	assert.Len(t, users10, 2)

	DB.Scopes(WhereGt("age", 22).Scope()).Find(&users11)
	assert.Len(t, users11, 0)

	DB.Scopes(WhereGt("age", 18).Scope()).Find(&users12)
	assert.Len(t, users12, 3)

	// Lt
	var users13, users14, users15 []User
	DB.Scopes(WhereLt("age", 20).Scope()).Find(&users13)
	assert.Len(t, users13, 1)

	DB.Scopes(WhereLt("age", 22).Scope()).Find(&users14)
	assert.Len(t, users14, 2)

	DB.Scopes(WhereLt("age", 18).Scope()).Find(&users15)
	assert.Len(t, users15, 0)

	// Ne
	var users16, users17, users18 []User
	DB.Scopes(WhereNe("age", 20).Scope()).Find(&users16)
	assert.Len(t, users16, 3)

	DB.Scopes(WhereNe("age", 22).Scope()).Find(&users17)
	assert.Len(t, users17, 2)

	DB.Scopes(WhereNe("age", 18).Scope()).Find(&users18)
	assert.Len(t, users18, 3)
}

func TestWhere_WhereNot(t *testing.T) {
	users := []*User{
		GetUser("WhereLikeUser1", GetUserOptions{Age: 18}),
		GetUser("WhereLikeUser2", GetUserOptions{Age: 20}),
		GetUser("WhereLikeUser3", GetUserOptions{Age: 22}),
	}

	CleanUsers()
	DB.Create(&users)

	var users1 []User
	DB.Scopes(WhereNot("name = ?", "WhereLikeUser1").Scope()).Find(&users1)
	assert.Len(t, users1, 2)
	assert.Equal(t, "WhereLikeUser2", users1[0].Name)
	assert.Equal(t, "WhereLikeUser3", users1[1].Name)
}

func TestWhere_Null(t *testing.T) {
	address2 := "WhereNullAddress2"
	address3 := "WhereNullAddress3"
	users := []*User{
		GetUser("WhereNullUser1", GetUserOptions{Age: 18, Address: nil}),
		GetUser("WhereNullUser2", GetUserOptions{Age: 20, Address: &address2}),
		GetUser("WhereNullUser3", GetUserOptions{Age: 22, Address: &address3}),
	}

	CleanUsers()
	DB.Create(&users)

	// Null
	var users1 []User
	DB.Scopes(WhereNull("address").Scope()).Find(&users1)
	assert.Len(t, users1, 1)
	assert.Equal(t, "WhereNullUser1", users1[0].Name)

	// NotNull
	var users2 []User
	DB.Scopes(WhereNotNull("address").Scope()).Find(&users2)
	assert.Len(t, users2, 2)
	assert.Equal(t, "WhereNullUser2", users2[0].Name)
	assert.Equal(t, "WhereNullUser3", users2[1].Name)
}
