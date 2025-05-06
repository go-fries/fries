package scope

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// OrderBy add order by condition
//
//	OrderBy("name")
//	OrderBy("name", "desc")
//	OrderBy("name", "asc")
func OrderBy(column string, reorder ...string) Scopes {
	return Scopes{}.OrderBy(column, reorder...)
}

// OrderByDesc add order by desc condition
//
//	OrderByDesc("name")
func OrderByDesc(column string) Scopes {
	return Scopes{}.OrderByDesc(column)
}

// OrderByAsc add order by asc condition
//
//	OrderByAsc("name")
func OrderByAsc(column string) Scopes {
	return Scopes{}.OrderByAsc(column)
}

// OrderByRaw add order by raw condition
//
//	OrderByRaw("name desc")
//	OrderByRaw("name asc")
//	OrderByRaw("name desc, age asc")
//	OrderByRaw("FIELD(id, 3, 1, 2)")
func OrderByRaw(sql any) Scopes {
	return Scopes{}.OrderByRaw(sql)
}

// OrderBy add order by condition
//
//	OrderBy("name")
//	OrderBy("name", "desc")
//	OrderBy("name", "asc")
func (s Scopes) OrderBy(column string, reorder ...string) Scopes {
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Order(fmt.Sprintf("%s %s", column, s.buildReorder(reorder...)))
	})
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
	return s.Add(func(db *gorm.DB) *gorm.DB {
		return db.Order(sql)
	})
}

func (s Scopes) buildReorder(reorder ...string) string {
	if len(reorder) > 0 && strings.ToUpper(reorder[0]) == "DESC" {
		return "DESC"
	}
	return "ASC"
}
