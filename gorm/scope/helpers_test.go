package scope

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestHelpers_When(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("WhenUser1", GetUserOptions{Age: 18}),
		GetUser("WhenUser2", GetUserOptions{Age: 20}),
		GetUser("WhenUser3", GetUserOptions{Age: 22}),
	}

	require.NoError(t, db.Create(&users).Error)

	// When
	t.Run("When", func(t *testing.T) {
		var users1, users2 []User
		require.NoError(t, db.Order("id").Scopes(When(true, func(db *gorm.DB) *gorm.DB {
			return db.Where("age > ?", 18)
		})).Find(&users1).Error)
		require.Len(t, users1, 2)
		assert.Equal(t, "WhenUser2", users1[0].Name)
		assert.Equal(t, "WhenUser3", users1[1].Name)

		require.NoError(t, db.Order("id").Scopes(When(false, func(db *gorm.DB) *gorm.DB {
			return db.Where("age > ?", 18)
		})).Find(&users2).Error)
		assert.Len(t, users2, 3)
	})

	// Unless
	t.Run("Unless", func(t *testing.T) {
		var users3, users4 []User
		require.NoError(t, db.Order("id").Scopes(Unless(true, func(db *gorm.DB) *gorm.DB {
			return db.Where("age > ?", 18)
		})).Find(&users3).Error)
		assert.Len(t, users3, 3)

		require.NoError(t, db.Order("id").Scopes(Unless(false, func(db *gorm.DB) *gorm.DB {
			return db.Where("age > ?", 18)
		})).Find(&users4).Error)
		require.Len(t, users4, 2)
		assert.Equal(t, "WhenUser2", users4[0].Name)
		assert.Equal(t, "WhenUser3", users4[1].Name)
	})

	// multiple When and Unless
	t.Run("multiple When and Unless", func(t *testing.T) {
		var users5 []User
		require.NoError(t, db.Order("id").Scopes(When(true, func(db *gorm.DB) *gorm.DB {
			return db.Where("age > ?", 18)
		}), When(false, func(db *gorm.DB) *gorm.DB {
			return db.Where("age < ?", 22)
		})).Find(&users5).Error)
		require.Len(t, users5, 2)
		assert.Equal(t, "WhenUser2", users5[0].Name)
	})
}

func TestHelpers_Where_Where(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("WhereUser1", GetUserOptions{}),
		GetUser("WhereUser2", GetUserOptions{}),
		GetUser("WhereUser3", GetUserOptions{}),
	}

	require.NoError(t, db.Create(&users).Error)

	var users1, users2, users3 []User
	require.NoError(t, db.Order("id").Scopes(Where("name in (?)", []string{"WhereUser1", "WhereUser2"})).Find(&users1).Error)
	assert.Len(t, users1, 2)

	require.NoError(t, db.Order("id").Scopes(Where("name in (?)", []string{"WhereUser1", "WhereUser4"})).Find(&users2).Error)
	assert.Len(t, users2, 1)

	require.NoError(t, db.Order("id").Scopes(Where("name = ?", "WhereUser3")).Find(&users3).Error)
	require.Len(t, users3, 1)
	assert.Equal(t, "WhereUser3", users3[0].Name)
}

func TestHelpers_WhereBetween(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("WhereBetweenUser1", GetUserOptions{Age: 18}),
		GetUser("WhereBetweenUser2", GetUserOptions{Age: 20}),
		GetUser("WhereBetweenUser3", GetUserOptions{Age: 22}),
	}

	require.NoError(t, db.Create(&users).Error)

	var users1, users2, users3 []User
	require.NoError(t, db.Order("id").Scopes(WhereBetween("age", 18, 20)).Find(&users1).Error)
	assert.Len(t, users1, 2)

	require.NoError(t, db.Order("id").Scopes(WhereBetween("age", 18, 19)).Find(&users2).Error)
	require.Len(t, users2, 1)
	assert.Equal(t, "WhereBetweenUser1", users2[0].Name)

	require.NoError(t, db.Order("id").Scopes(WhereBetween("age", 12, 16)).Find(&users3).Error)
	assert.Len(t, users3, 0)

	var users4, users5, users6 []User
	require.NoError(t, db.Order("id").Scopes(WhereNotBetween("age", 18, 20)).Find(&users4).Error)
	require.Len(t, users4, 1)
	assert.Equal(t, "WhereBetweenUser3", users4[0].Name)

	require.NoError(t, db.Order("id").Scopes(WhereNotBetween("age", 18, 19)).Find(&users5).Error)
	assert.Len(t, users5, 2)

	require.NoError(t, db.Order("id").Scopes(WhereNotBetween("age", 12, 16)).Find(&users6).Error)
	assert.Len(t, users6, 3)
}

func TestHelpers_WhereIn(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("WhereInUser1", GetUserOptions{Age: 18}),
		GetUser("WhereInUser2", GetUserOptions{Age: 20}),
		GetUser("WhereInUser3", GetUserOptions{Age: 22}),
	}

	require.NoError(t, db.Create(&users).Error)

	var users1, users2, users3 []User
	require.NoError(t, db.Order("id").Scopes(WhereIn("name", "WhereInUser1", "WhereInUser2")).Find(&users1).Error)
	assert.Len(t, users1, 2)

	require.NoError(t, db.Order("id").Scopes(WhereIn("age", []int{18, 20})).Find(&users2).Error)
	assert.Len(t, users2, 2)

	require.NoError(t, db.Order("id").Scopes(WhereIn("name", []string{"WhereInUser1", "WhereInUser2"})).Find(&users3).Error)
	assert.Len(t, users3, 2)

	var users4, users5, users6 []User
	require.NoError(t, db.Order("id").Scopes(WhereNotIn("name", "WhereInUser1", "WhereInUser2")).Find(&users4).Error)
	require.Len(t, users4, 1)
	assert.Equal(t, "WhereInUser3", users4[0].Name)

	require.NoError(t, db.Order("id").Scopes(WhereNotIn("age", []int{18, 20})).Find(&users5).Error)
	require.Len(t, users5, 1)
	assert.Equal(t, "WhereInUser3", users5[0].Name)

	require.NoError(t, db.Order("id").Scopes(WhereNotIn("name", []string{"WhereInUser1", "WhereInUser2"})).Find(&users6).Error)
	require.Len(t, users6, 1)
	assert.Equal(t, "WhereInUser3", users6[0].Name)
}

func TestHelpers_WhereLike(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("WhereLikeUser1", GetUserOptions{Age: 18}),
		GetUser("WhereLikeUser2", GetUserOptions{Age: 20}),
		GetUser("WhereLikeUser3", GetUserOptions{Age: 22}),
	}

	require.NoError(t, db.Create(&users).Error)

	var users1, users2, users3, users4 []User
	require.NoError(t, db.Order("id").Scopes(WhereLike("name", "WhereLikeUser1")).Find(&users1).Error)
	require.Len(t, users1, 1)
	assert.Equal(t, "WhereLikeUser1", users1[0].Name)

	require.NoError(t, db.Order("id").Scopes(WhereLike("name", "WhereLike%")).Find(&users2).Error)
	assert.Len(t, users2, 3)

	require.NoError(t, db.Order("id").Scopes(WhereLike("name", "%LikeUser3")).Find(&users3).Error)
	require.Len(t, users3, 1)
	assert.Equal(t, "WhereLikeUser3", users3[0].Name)

	require.NoError(t, db.Order("id").Scopes(WhereLike("name", "%Like%")).Find(&users4).Error)
	assert.Len(t, users4, 3)

	var users5, users6, users7, users8 []User
	require.NoError(t, db.Order("id").Scopes(WhereNotLike("name", "WhereLikeUser1")).Find(&users5).Error)
	assert.Len(t, users5, 2)

	require.NoError(t, db.Order("id").Scopes(WhereNotLike("name", "WhereLike%")).Find(&users6).Error)
	assert.Len(t, users6, 0)

	require.NoError(t, db.Order("id").Scopes(WhereNotLike("name", "%LikeUser3")).Find(&users7).Error)
	assert.Len(t, users7, 2)

	require.NoError(t, db.Order("id").Scopes(WhereNotLike("name", "%Like%")).Find(&users8).Error)
	assert.Len(t, users8, 0)
}

func TestHelpers_WhereOP(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("WhereLikeUser1", GetUserOptions{Age: 18}),
		GetUser("WhereLikeUser2", GetUserOptions{Age: 20}),
		GetUser("WhereLikeUser3", GetUserOptions{Age: 22}),
		GetUser("WhereLikeUser4", GetUserOptions{Age: 22}),
	}

	require.NoError(t, db.Create(&users).Error)

	// Eq
	var users1, users2, users3 []User
	require.NoError(t, db.Order("id").Scopes(WhereEq("name", "WhereLikeUser1")).Find(&users1).Error)
	require.Len(t, users1, 1)
	assert.Equal(t, "WhereLikeUser1", users1[0].Name)

	require.NoError(t, db.Order("id").Scopes(WhereEq("age", 18)).Find(&users2).Error)
	require.Len(t, users2, 1)
	assert.Equal(t, "WhereLikeUser1", users2[0].Name)

	require.NoError(t, db.Order("id").Scopes(WhereEq("age", 22)).Find(&users3).Error)
	assert.Len(t, users3, 2)

	// Egt
	var users4, users5, users6 []User
	require.NoError(t, db.Order("id").Scopes(WhereEgt("age", 20)).Find(&users4).Error)
	assert.Len(t, users4, 3)

	require.NoError(t, db.Order("id").Scopes(WhereEgt("age", 22)).Find(&users5).Error)
	assert.Len(t, users5, 2)

	require.NoError(t, db.Order("id").Scopes(WhereEgt("age", 23)).Find(&users6).Error)
	assert.Len(t, users6, 0)

	// Elt
	var users7, users8, users9 []User
	require.NoError(t, db.Order("id").Scopes(WhereElt("age", 20)).Find(&users7).Error)
	assert.Len(t, users7, 2)

	require.NoError(t, db.Order("id").Scopes(WhereElt("age", 22)).Find(&users8).Error)
	assert.Len(t, users8, 4)

	require.NoError(t, db.Order("id").Scopes(WhereElt("age", 18)).Find(&users9).Error)
	assert.Len(t, users9, 1)

	// Gt
	var users10, users11, users12 []User
	require.NoError(t, db.Order("id").Scopes(WhereGt("age", 20)).Find(&users10).Error)
	assert.Len(t, users10, 2)

	require.NoError(t, db.Order("id").Scopes(WhereGt("age", 22)).Find(&users11).Error)
	assert.Len(t, users11, 0)

	require.NoError(t, db.Order("id").Scopes(WhereGt("age", 18)).Find(&users12).Error)
	assert.Len(t, users12, 3)

	// Lt
	var users13, users14, users15 []User
	require.NoError(t, db.Order("id").Scopes(WhereLt("age", 20)).Find(&users13).Error)
	assert.Len(t, users13, 1)

	require.NoError(t, db.Order("id").Scopes(WhereLt("age", 22)).Find(&users14).Error)
	assert.Len(t, users14, 2)

	require.NoError(t, db.Order("id").Scopes(WhereLt("age", 18)).Find(&users15).Error)
	assert.Len(t, users15, 0)

	// Ne
	var users16, users17, users18 []User
	require.NoError(t, db.Order("id").Scopes(WhereNe("age", 20)).Find(&users16).Error)
	assert.Len(t, users16, 3)

	require.NoError(t, db.Order("id").Scopes(WhereNe("age", 22)).Find(&users17).Error)
	assert.Len(t, users17, 2)

	require.NoError(t, db.Order("id").Scopes(WhereNe("age", 18)).Find(&users18).Error)
	assert.Len(t, users18, 3)
}

func TestHelpers_WhereNot(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("WhereLikeUser1", GetUserOptions{Age: 18}),
		GetUser("WhereLikeUser2", GetUserOptions{Age: 20}),
		GetUser("WhereLikeUser3", GetUserOptions{Age: 22}),
	}

	require.NoError(t, db.Create(&users).Error)

	var users1 []User
	require.NoError(t, db.Order("id").Scopes(WhereNot("name = ?", "WhereLikeUser1")).Find(&users1).Error)
	require.Len(t, users1, 2)
	assert.Equal(t, "WhereLikeUser2", users1[0].Name)
	assert.Equal(t, "WhereLikeUser3", users1[1].Name)
}

func TestHelpers_WhereNullAndNotNull(t *testing.T) {
	db := newTestDB(t)
	address2 := "WhereNullAddress2"
	address3 := "WhereNullAddress3"
	users := []*User{
		GetUser("WhereNullUser1", GetUserOptions{Age: 18, Address: nil}),
		GetUser("WhereNullUser2", GetUserOptions{Age: 20, Address: &address2}),
		GetUser("WhereNullUser3", GetUserOptions{Age: 22, Address: &address3}),
	}

	require.NoError(t, db.Create(&users).Error)

	// Null
	var users1 []User
	require.NoError(t, db.Order("id").Scopes(WhereNull("address")).Find(&users1).Error)
	require.Len(t, users1, 1)
	assert.Equal(t, "WhereNullUser1", users1[0].Name)

	// NotNull
	var users2 []User
	require.NoError(t, db.Order("id").Scopes(WhereNotNull("address")).Find(&users2).Error)
	require.Len(t, users2, 2)
	assert.Equal(t, "WhereNullUser2", users2[0].Name)
	assert.Equal(t, "WhereNullUser3", users2[1].Name)
}

func TestPagination(t *testing.T) {
	db := newTestDB(t)
	users := []*User{
		GetUser("PaginationUser1", GetUserOptions{}),
		GetUser("PaginationUser2", GetUserOptions{}),
		GetUser("PaginationUser3", GetUserOptions{}),
		GetUser("PaginationUser4", GetUserOptions{}),
		GetUser("PaginationUser5", GetUserOptions{}),
	}

	require.NoError(t, db.Create(&users).Error)

	var users1, users2, users3, users4, users5, users6 []User

	// Offset&Limit
	require.NoError(t, db.Order("id").Scopes(Offset(2), Limit(2)).Find(&users1).Error)
	require.Len(t, users1, 2)
	assert.Equal(t, "PaginationUser3", users1[0].Name)
	assert.Equal(t, "PaginationUser4", users1[1].Name)

	require.NoError(t, db.Order("id").Scopes(Limit(2), Skip(4)).Find(&users2).Error)
	require.Len(t, users2, 1)
	assert.Equal(t, "PaginationUser5", users2[0].Name)

	// Skip&Take
	require.NoError(t, db.Order("id").Scopes(Skip(2), Take(2)).Find(&users3).Error)
	require.Len(t, users3, 2)
	assert.Equal(t, "PaginationUser3", users3[0].Name)
	assert.Equal(t, "PaginationUser4", users3[1].Name)

	require.NoError(t, db.Order("id").Scopes(Take(2), Skip(4)).Find(&users4).Error)
	require.Len(t, users4, 1)
	assert.Equal(t, "PaginationUser5", users4[0].Name)

	// Page
	require.NoError(t, db.Order("id").Scopes(Page(2, 2)).Find(&users5).Error)
	require.Len(t, users5, 2)
	assert.Equal(t, "PaginationUser3", users5[0].Name)

	require.NoError(t, db.Order("id").Scopes(Page(3, 2)).Find(&users6).Error)
	require.Len(t, users6, 1)
	assert.Equal(t, "PaginationUser5", users6[0].Name)
}

func TestHelpers_OrderBy(t *testing.T) {
	db := newTestDB(t)
	birthday1 := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	birthday2 := birthday1
	birthday3 := birthday1.Add(2 * time.Hour)
	birthday4 := birthday1.Add(3 * time.Hour)
	users := []*User{
		GetUser("OrderUser1", GetUserOptions{Age: 17, Birthday: &birthday1}),
		GetUser("OrderUser2", GetUserOptions{Age: 20, Birthday: &birthday2}),
		GetUser("OrderUser3", GetUserOptions{Age: 21, Birthday: &birthday3}),
		GetUser("OrderUser4", GetUserOptions{Age: 22, Birthday: &birthday4}),
	}

	require.NoError(t, db.Create(&users).Error)

	// OrderBy
	var users1, users2, users3, users4 []User
	require.NoError(t, db.Scopes(OrderBy("age")).Limit(2).Find(&users1).Error)
	require.Len(t, users1, 2)
	assert.Equal(t, "OrderUser1", users1[0].Name)
	assert.Equal(t, "OrderUser2", users1[1].Name)

	require.NoError(t, db.Scopes(OrderBy("age", "asc")).Limit(2).Find(&users2).Error)
	require.Len(t, users2, 2)
	assert.Equal(t, "OrderUser1", users2[0].Name)
	assert.Equal(t, "OrderUser2", users2[1].Name)

	require.NoError(t, db.Scopes(OrderBy("age", "desc")).Limit(2).Find(&users3).Error)
	require.Len(t, users3, 2)
	assert.Equal(t, "OrderUser4", users3[0].Name)
	assert.Equal(t, "OrderUser3", users3[1].Name)

	require.NoError(t, db.Scopes(OrderBy("age", "unknown")).Limit(2).Find(&users4).Error)
	require.Len(t, users4, 2)
	assert.Equal(t, "OrderUser1", users4[0].Name)
	assert.Equal(t, "OrderUser2", users4[1].Name)

	// OrderByAsc
	var users5 []User
	require.NoError(t, db.Scopes(OrderByAsc("age")).Limit(2).Find(&users5).Error)
	require.Len(t, users5, 2)
	assert.Equal(t, "OrderUser1", users5[0].Name)
	assert.Equal(t, "OrderUser2", users5[1].Name)

	// OrderByDesc
	var users6 []User
	require.NoError(t, db.Scopes(OrderByDesc("age")).Limit(2).Find(&users6).Error)
	require.Len(t, users6, 2)
	assert.Equal(t, "OrderUser4", users6[0].Name)
	assert.Equal(t, "OrderUser3", users6[1].Name)

	// OrderByRaw
	var users7 []User
	require.NoError(t, db.Scopes(OrderByRaw("age % 2 asc"), OrderBy("id")).Limit(2).Find(&users7).Error)
	require.Len(t, users7, 2)
	assert.Equal(t, "OrderUser2", users7[0].Name)
	assert.Equal(t, "OrderUser4", users7[1].Name)

	// multiple OrderBy
	var users8 []User
	require.NoError(t, db.Scopes(OrderBy("birthday", "asc"), OrderBy("age", "desc")).Limit(2).Find(&users8).Error)
	require.Len(t, users8, 2)
	assert.Equal(t, "OrderUser2", users8[0].Name)
	assert.Equal(t, "OrderUser1", users8[1].Name)
}
