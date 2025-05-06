package scope

import (
	"gorm.io/gorm"
)

type Scopes []func(*gorm.DB) *gorm.DB

func (s Scopes) Apply(db *gorm.DB) *gorm.DB {
	return db.Scopes(s...)
}

func (s Scopes) Scope() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return s.Apply(db)
	}
}

func (s Scopes) Scopes() []func(*gorm.DB) *gorm.DB {
	return s
}

func (s Scopes) Add(scopes ...func(*gorm.DB) *gorm.DB) Scopes {
	return append(s, scopes...)
}

// ==============================================================================
// Trait
// ==============================================================================

// When if condition is true, apply the scopes
//
//	When(true, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
//	When(false, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
func (s Scopes) When(condition bool, fc func(*gorm.DB) *gorm.DB) Scopes {
	return s.Add(When(condition, fc))
}

// Unless if condition is false, apply the scopes
//
//	Unless(false, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
//	Unless(true, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
func (s Scopes) Unless(condition bool, fc func(*gorm.DB) *gorm.DB) Scopes {
	return s.Add(Unless(condition, fc))
}

// ==============================================================================
// Order
// ==============================================================================

// OrderBy add order by condition
//
//	OrderBy("name")
//	OrderBy("name", "desc")
//	OrderBy("name", "asc")
func (s Scopes) OrderBy(column string, reorder ...string) Scopes {
	return s.Add(OrderBy(column, reorder...))
}

// OrderByDesc add order by desc condition
//
//	OrderByDesc("name")
func (s Scopes) OrderByDesc(column string) Scopes {
	return s.OrderBy(column, "desc")
}

// OrderByAsc add order by asc condition
//
//	OrderByAsc("name")
func (s Scopes) OrderByAsc(column string) Scopes {
	return s.OrderBy(column, "asc")
}

// OrderByRaw add order by raw condition
//
//	OrderByRaw("name desc")
//	OrderByRaw("name asc")
//	OrderByRaw("name desc, age asc")
//	OrderByRaw("FIELD(id, 3, 1, 2)")
func (s Scopes) OrderByRaw(sql any) Scopes {
	return s.Add(OrderByRaw(sql))
}

// ==============================================================================
// Offset/Limit/Page
// ==============================================================================

// Offset add offset condition
//
//	Offset(3)
func (s Scopes) Offset(offset int) Scopes {
	return s.Add(Offset(offset))
}

// Skip add offset condition
//
//	Skip(3)
func (s Scopes) Skip(offset int) Scopes {
	return s.Offset(offset)
}

// Limit add limit condition
//
//	Limit(3)
func (s Scopes) Limit(limit int) Scopes {
	return s.Add(Limit(limit))
}

// Take add limit condition
//
//	Take(3)
func (s Scopes) Take(limit int) Scopes {
	return s.Limit(limit)
}

// Page add page condition
//
//	Page(2, 10)
func (s Scopes) Page(page, prePage int) Scopes {
	return s.Add(Page(page, prePage))
}

// ==============================================================================
// Where
// ==============================================================================

// Where add where condition
//
//	Where("name = ?", "Flc")
//	Where("name = ? AND age = ?", "Flc", 20)
func (s Scopes) Where(query any, args ...any) Scopes {
	return s.Add(Where(query, args...))
}

// WhereBetween add where between condition
//
//	WhereBetween("age", 18, 20)
func (s Scopes) WhereBetween(field string, start, end any) Scopes {
	return s.Add(WhereBetween(field, start, end))
}

// WhereNotBetween add where not between condition
//
//	WhereNotBetween("age", 18, 20)
func (s Scopes) WhereNotBetween(field string, start, end any) Scopes {
	return s.Add(WhereNotBetween(field, start, end))
}

// WhereIn add where in condition
//
//	WhereIn("name", []string{"WhereInUser1", "WhereInUser2"})
//	WhereIn("age", []int{18, 20})
//	WhereIn("name", "WhereInUser1", "WhereInUser2")
func (s Scopes) WhereIn(field string, values ...any) Scopes {
	return s.Add(WhereIn(field, values...))
}

// WhereNotIn add where not in condition
//
//	WhereNotIn("name", []string{"WhereInUser1", "WhereInUser2"})
//	WhereNotIn("age", []int{18, 20})
//	WhereNotIn("name", "WhereInUser1", "WhereInUser2")
func (s Scopes) WhereNotIn(field string, values ...any) Scopes {
	return s.Add(WhereNotIn(field, values...))
}

// WhereLike add where like condition
//
//	WhereLike("name", "Flc")
//	WhereLike("name", "Flc%")
//	WhereLike("name", "%Flc")
//	WhereLike("name", "%Flc%")
func (s Scopes) WhereLike(field string, value any) Scopes {
	return s.Add(WhereLike(field, value))
}

// WhereNotLike add where not like condition
//
//	WhereNotLike("name", "Flc")
//	WhereNotLike("name", "Flc%")
//	WhereNotLike("name", "%Flc")
//	WhereNotLike("name", "%Flc%")
func (s Scopes) WhereNotLike(field string, value any) Scopes {
	return s.Add(WhereNotLike(field, value))
}

// WhereEq add where eq condition
//
//	WhereEq("name", "Flc")
//	WhereEq("age", 18)
func (s Scopes) WhereEq(field string, value any) Scopes {
	return s.Add(WhereEq(field, value))
}

// WhereEgt add where egt condition
//
//	WhereEgt("age", 18)
func (s Scopes) WhereEgt(field string, value any) Scopes {
	return s.Add(WhereEgt(field, value))
}

// WhereGt add where gt condition
//
//	WhereGt("age", 18)
func (s Scopes) WhereGt(field string, value any) Scopes {
	return s.Add(WhereGt(field, value))
}

// WhereElt add where elt condition
//
//	WhereElt("age", 18)
func (s Scopes) WhereElt(field string, value any) Scopes {
	return s.Add(WhereElt(field, value))
}

// WhereLt add where lt condition
//
//	WhereLt("age", 18)
func (s Scopes) WhereLt(field string, value any) Scopes {
	return s.Add(WhereLt(field, value))
}

// WhereNe add where ne condition
//
//	WhereNe("name", "Flc")
//	WhereNe("age", 18)
func (s Scopes) WhereNe(field string, value any) Scopes {
	return s.Add(WhereNe(field, value))
}

// WhereNot add where not condition
//
//	WhereNot("name = ?", "Flc")
//	WhereNot("name = ? AND age = ?", "Flc", 20)
func (s Scopes) WhereNot(query any, args ...any) Scopes {
	return s.Add(WhereNot(query, args...))
}

// WhereNull add where null condition
//
//	WhereNull("name")
func (s Scopes) WhereNull(field string) Scopes {
	return s.Add(WhereNull(field))
}

// WhereNotNull add where not null condition
//
//	WhereNotNull("name")
func (s Scopes) WhereNotNull(field string) Scopes {
	return s.Add(WhereNotNull(field))
}
