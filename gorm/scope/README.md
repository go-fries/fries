# Gorm/Scopes

## Example

```go
package main

import (
	"time"

	"gorm.io/gorm"

	"github.com/go-kratos-ecosystem/components/v2/gorm/scopes"
)

func main() {
	var db *gorm.DB

	db.Scopes(scopes.
		// trait
		When(true, func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}).
		Unless(true, func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NOT NULL")
		}).

		// Where
		Where("name = ?", "Flc").
		WhereBetween("created_at", time.Now(), time.Now()).
		WhereNotBetween("created_at", time.Now(), time.Now()).
		WhereIn("name", "Flc", "Flc 2").
		WhereNotIn("name", "Flc", "Flc 2").
		WhereLike("name", "Flc%").
		WhereNotLike("name", "Flc%").
		WhereEq("name", "Flc").
		WhereNe("name", "Flc").
		WhereGt("age", 18).
		WhereEgt("age", 18).
		WhereLt("age", 18).
		WhereElt("age", 18).

		// Order
		OrderBy("id").
		OrderBy("id", "desc").
		OrderBy("id", "asc").
		OrderByDesc("id").
		OrderByAsc("id").
		OrderByRaw("id desc").

		// Limit
		Limit(10).
		Take(10).

		// Offset
		Offset(10).
		Skip(10).

		// Page
		Page(1, 20).

		// To Scope()
		Scope()).
		Find(&[]struct{}{})
}
```