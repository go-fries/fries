package scope

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ==============================================================================
// Trait
// ==============================================================================

// When if condition is true, apply the scopes
//
//	When(true, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
//	When(false, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
func When(condition bool, f func(db *gorm.DB) *gorm.DB) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if condition {
			return f(db)
		}
		return db
	}
}

// Unless if condition is false, apply the scopes
//
//	Unless(false, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
//	Unless(true, func(db *gorm.DB) *gorm.DB { return db.Where("name = ?", "Flc") })
func Unless(condition bool, f func(db *gorm.DB) *gorm.DB) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if !condition {
			return f(db)
		}
		return db
	}
}

// ==============================================================================
// Order
// ==============================================================================

// OrderBy add order by condition
//
//	OrderBy("name")
//	OrderBy("name", "desc")
//	OrderBy("name", "asc")
func OrderBy(column string, reorder ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(fmt.Sprintf("%s %s", column, buildReorder(reorder...)))
	}
}

func buildReorder(reorder ...string) string {
	if len(reorder) > 0 && strings.ToUpper(reorder[0]) == "DESC" {
		return "DESC"
	}
	return "ASC"
}

// OrderByDesc add order by desc condition
//
//	OrderByDesc("name")
func OrderByDesc(column string) func(db *gorm.DB) *gorm.DB {
	return OrderBy(column, "desc")
}

// OrderByAsc add order by asc condition
//
//	OrderByAsc("name")
func OrderByAsc(column string) func(db *gorm.DB) *gorm.DB {
	return OrderBy(column, "asc")
}

// OrderByRaw add order by raw condition
//
//	OrderByRaw("name desc")
//	OrderByRaw("name asc")
//	OrderByRaw("name desc, age asc")
//	OrderByRaw("FIELD(id, 3, 1, 2)")
func OrderByRaw(sql any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(sql)
	}
}

// ==============================================================================
// Pagination: Offset/Limit/Page
// ==============================================================================

// Offset add offset condition
//
//	Offset(3)
func Offset(offset int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset)
	}
}

// Skip add offset condition
//
//	Skip(3)
func Skip(offset int) func(db *gorm.DB) *gorm.DB {
	return Offset(offset)
}

// Limit add limit condition
//
//	Limit(3)
func Limit(limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	}
}

// Take add limit condition
//
//	Take(3)
func Take(limit int) func(db *gorm.DB) *gorm.DB {
	return Limit(limit)
}

// Page add page condition
//
//	Page(2, 10)
func Page(page, prePage int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset((page - 1) * prePage).Limit(prePage)
	}
}

// ==============================================================================
// Where
// ==============================================================================

// Where add where condition
//
//	Where("name = ?", "Flc")
//	Where("name = ? AND age = ?", "Flc", 20)
func Where(query any, args ...any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(query, args...)
	}
}

// WhereBetween add where between condition
//
//	WhereBetween("age", 18, 20)
func WhereBetween(field string, start, end any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", field), start, end)
	}
}

// WhereNotBetween add where not between condition
//
//	WhereNotBetween("age", 18, 20)
func WhereNotBetween(field string, start, end any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s NOT BETWEEN ? AND ?", field), start, end)
	}
}

// WhereIn add where in condition
//
//	WhereIn("name", []string{"WhereInUser1", "WhereInUser2"})
//	WhereIn("age", []int{18, 20})
//	WhereIn("name", "WhereInUser1", "WhereInUser2")
func WhereIn(field string, values ...any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(values) > 1 {
			return db.Where(fmt.Sprintf("%s IN (?)", field), values)
		}
		return db.Where(fmt.Sprintf("%s IN ?", field), values...)
	}
}

// WhereNotIn add where not in condition
//
//	WhereNotIn("name", []string{"WhereInUser1", "WhereInUser2"})
//	WhereNotIn("age", []int{18, 20})
//	WhereNotIn("name", "WhereInUser1", "WhereInUser2")
func WhereNotIn(field string, values ...any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(values) > 1 {
			return db.Where(fmt.Sprintf("%s NOT IN (?)", field), values)
		}
		return db.Where(fmt.Sprintf("%s NOT IN ?", field), values...)
	}
}

// WhereLike add where like condition
//
//	WhereLike("name", "Flc")
//	WhereLike("name", "Flc%")
//	WhereLike("name", "%Flc")
//	WhereLike("name", "%Flc%")
func WhereLike(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s LIKE ?", field), value)
	}
}

// WhereNotLike add where not like condition
//
//	WhereNotLike("name", "Flc")
//	WhereNotLike("name", "Flc%")
//	WhereNotLike("name", "%Flc")
//	WhereNotLike("name", "%Flc%")
func WhereNotLike(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s NOT LIKE ?", field), value)
	}
}

// WhereEq add where eq condition
//
//	WhereEq("name", "Flc")
//	WhereEq("age", 18)
func WhereEq(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s = ?", field), value)
	}
}

// WhereEgt add where egt condition
//
//	WhereEgt("age", 18)
func WhereEgt(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s >= ?", field), value)
	}
}

// WhereGt add where gt condition
//
//	WhereGt("age", 18)
func WhereGt(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s > ?", field), value)
	}
}

// WhereElt add where elt condition
//
//	WhereElt("age", 18)
func WhereElt(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s <= ?", field), value)
	}
}

// WhereLt add where lt condition
//
//	WhereLt("age", 18)
func WhereLt(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s < ?", field), value)
	}
}

// WhereNe add where ne condition
//
//	WhereNe("name", "Flc")
//	WhereNe("age", 18)
func WhereNe(field string, value any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s != ?", field), value)
	}
}

// WhereNot add where not condition
//
//	WhereNot("name = ?", "Flc")
//	WhereNot("name = ? AND age = ?", "Flc", 20)
func WhereNot(query any, args ...any) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Not(query, args...)
	}
}

// WhereNull add where null condition
//
//	WhereNull("name")
func WhereNull(field string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s IS NULL", field))
	}
}

// WhereNotNull add where not null condition
//
//	WhereNotNull("name")
func WhereNotNull(field string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s IS NOT NULL", field))
	}
}
