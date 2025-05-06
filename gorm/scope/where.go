package scope

import (
	"fmt"

	"gorm.io/gorm"
)

// Where add where condition
//
//	Where("name = ?", "Flc")
//	Where("name = ? AND age = ?", "Flc", 20)
func Where(query any, args ...any) Scopes {
	return Scopes{}.Where(query, args...)
}

// WhereBetween add where between condition
//
//	WhereBetween("age", 18, 20)
func WhereBetween(field string, start, end any) Scopes {
	return Scopes{}.WhereBetween(field, start, end)
}

// WhereNotBetween add where not between condition
//
//	WhereNotBetween("age", 18, 20)
func WhereNotBetween(field string, start, end any) Scopes {
	return Scopes{}.WhereNotBetween(field, start, end)
}

// WhereIn add where in condition
//
//	WhereIn("name", []string{"WhereInUser1", "WhereInUser2"})
//	WhereIn("age", []int{18, 20})
//	WhereIn("name", "WhereInUser1", "WhereInUser2")
func WhereIn(field string, values ...any) Scopes {
	return Scopes{}.WhereIn(field, values...)
}

// WhereNotIn add where not in condition
//
//	WhereNotIn("name", []string{"WhereInUser1", "WhereInUser2"})
//	WhereNotIn("age", []int{18, 20})
//	WhereNotIn("name", "WhereInUser1", "WhereInUser2")
func WhereNotIn(field string, values ...any) Scopes {
	return Scopes{}.WhereNotIn(field, values...)
}

// WhereLike add where like condition
//
//	WhereLike("name", "Flc")
//	WhereLike("name", "Flc%")
//	WhereLike("name", "%Flc")
//	WhereLike("name", "%Flc%")
func WhereLike(field string, value any) Scopes {
	return Scopes{}.WhereLike(field, value)
}

// WhereNotLike add where not like condition
//
//	WhereNotLike("name", "Flc")
//	WhereNotLike("name", "Flc%")
//	WhereNotLike("name", "%Flc")
//	WhereNotLike("name", "%Flc%")
func WhereNotLike(field string, value any) Scopes {
	return Scopes{}.WhereNotLike(field, value)
}

// WhereEq add where eq condition
//
//	WhereEq("name", "Flc")
//	WhereEq("age", 18)
func WhereEq(field string, value any) Scopes {
	return Scopes{}.WhereEq(field, value)
}

// WhereEgt add where egt condition
//
//	WhereEgt("age", 18)
func WhereEgt(field string, value any) Scopes {
	return Scopes{}.WhereEgt(field, value)
}

// WhereGt add where gt condition
//
//	WhereGt("age", 18)
func WhereGt(field string, value any) Scopes {
	return Scopes{}.WhereGt(field, value)
}

// WhereElt add where elt condition
//
//	WhereElt("age", 18)
func WhereElt(field string, value any) Scopes {
	return Scopes{}.WhereElt(field, value)
}

// WhereLt add where lt condition
//
//	WhereLt("age", 18)
func WhereLt(field string, value any) Scopes {
	return Scopes{}.WhereLt(field, value)
}

// WhereNe add where ne condition
//
//	WhereNe("name", "Flc")
//	WhereNe("age", 18)
func WhereNe(field string, value any) Scopes {
	return Scopes{}.WhereNe(field, value)
}

// WhereNot add where not condition
//
//	WhereNot("name = ?", "Flc")
//	WhereNot("name = ? AND age = ?", "Flc", 20)
func WhereNot(query any, args ...any) Scopes {
	return Scopes{}.WhereNot(query, args...)
}

// WhereNull add where null condition
//
//	WhereNull("name")
func WhereNull(field string) Scopes {
	return Scopes{}.WhereNull(field)
}

// WhereNotNull add where not null condition
//
//	WhereNotNull("name")
func WhereNotNull(field string) Scopes {
	return Scopes{}.WhereNotNull(field)
}

// Where add where condition
//
//	Where("name = ?", "Flc")
//	Where("name = ? AND age = ?", "Flc", 20)
func (s Scopes) Where(query any, args ...any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(query, args...)
	})
}

// WhereBetween add where between condition
//
//	WhereBetween("age", 18, 20)
func (s Scopes) WhereBetween(column string, start, end any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", column), start, end)
	})
}

// WhereNotBetween add where not between condition
//
//	WhereNotBetween("age", 18, 20)
func (s Scopes) WhereNotBetween(column string, start, end any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s NOT BETWEEN ? AND ?", column), start, end)
	})
}

// WhereIn add where in condition
//
//	WhereIn("name", []string{"WhereInUser1", "WhereInUser2"})
//	WhereIn("age", []int{18, 20})
//	WhereIn("name", "WhereInUser1", "WhereInUser2")
func (s Scopes) WhereIn(column string, values ...any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		if len(values) > 1 {
			return db.Where(fmt.Sprintf("%s IN (?)", column), values)
		}
		return db.Where(fmt.Sprintf("%s IN ?", column), values...)
	})
}

// WhereNotIn add where not in condition
//
//	WhereNotIn("name", []string{"WhereInUser1", "WhereInUser2"})
//	WhereNotIn("age", []int{18, 20})
//	WhereNotIn("name", "WhereInUser1", "WhereInUser2")
func (s Scopes) WhereNotIn(column string, values ...any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		if len(values) > 1 {
			return db.Where(fmt.Sprintf("%s NOT IN (?)", column), values)
		}
		return db.Where(fmt.Sprintf("%s NOT IN ?", column), values...)
	})
}

// WhereLike add where like condition
//
//	WhereLike("name", "Flc")
//	WhereLike("name", "Flc%")
//	WhereLike("name", "%Flc")
//	WhereLike("name", "%Flc%")
func (s Scopes) WhereLike(column string, value any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s LIKE ?", column), value)
	})
}

// WhereNotLike add where not like condition
//
//	WhereNotLike("name", "Flc")
//	WhereNotLike("name", "Flc%")
//	WhereNotLike("name", "%Flc")
//	WhereNotLike("name", "%Flc%")
func (s Scopes) WhereNotLike(column string, value any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s NOT LIKE ?", column), value)
	})
}

// WhereEq add where eq condition
//
//	WhereEq("name", "Flc")
//	WhereEq("age", 18)
func (s Scopes) WhereEq(column string, value any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s = ?", column), value)
	})
}

// WhereEgt add where egt condition
//
//	WhereEgt("age", 18)
func (s Scopes) WhereEgt(column string, value any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s >= ?", column), value)
	})
}

// WhereGt add where gt condition
//
//	WhereGt("age", 18)
func (s Scopes) WhereGt(column string, value any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s > ?", column), value)
	})
}

// WhereElt add where elt condition
//
//	WhereElt("age", 18)
func (s Scopes) WhereElt(column string, value any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s <= ?", column), value)
	})
}

// WhereLt add where lt condition
//
//	WhereLt("age", 18)
func (s Scopes) WhereLt(column string, value any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s < ?", column), value)
	})
}

// WhereNe add where ne condition
//
//	WhereNe("name", "Flc")
//	WhereNe("age", 18)
func (s Scopes) WhereNe(column string, value any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s <> ?", column), value)
	})
}

// WhereNot add where not condition
//
//	WhereNot("name = ?", "Flc")
//	WhereNot("name = ? AND age = ?", "Flc", 20)
func (s Scopes) WhereNot(query any, args ...any) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Not(query, args...)
	})
}

// WhereNull add where null condition
//
//	WhereNull("name")
func (s Scopes) WhereNull(column string) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s IS NULL", column))
	})
}

// WhereNotNull add where not null condition
//
//	WhereNotNull("name")
func (s Scopes) WhereNotNull(column string) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s IS NOT NULL", column))
	})
}
