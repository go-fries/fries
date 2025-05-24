# Gorm/Scopes

## Example

```go
package scope_test

import (
	"time"

	"github.com/go-fries/fries/gorm/scope/v3"
	"gorm.io/gorm"
)

func Example_scopes() {
	var db *gorm.DB

	db.Scopes(
		// when
		scope.When(true, func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		}),
		scope.Unless(true, func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NOT NULL")
		}),

		// Where
		scope.Where("name = ?", "Flc"),
		scope.WhereBetween("created_at", time.Now(), time.Now()),
		scope.WhereNotBetween("created_at", time.Now(), time.Now()),
		scope.WhereIn("name", "Flc", "Flc 2"),
		scope.WhereNotIn("name", "Flc", "Flc 2"),
		scope.WhereLike("name", "Flc%"),
		scope.WhereNotLike("name", "Flc%"),
		scope.WhereEq("name", "Flc"),
		scope.WhereNe("name", "Flc"),
		scope.WhereGt("age", 18),
		scope.WhereEgt("age", 18),
		scope.WhereLt("age", 18),
		scope.WhereElt("age", 18),

		// Order
		scope.OrderBy("id"),
		scope.OrderBy("id", "desc"),
		scope.OrderBy("id", "asc"),
		scope.OrderByDesc("id"),
		scope.OrderByAsc("id"),
		scope.OrderByRaw("id desc"),

		// Limit
		scope.Limit(10),
		scope.Take(10),

		// Offset
		scope.Offset(10),
		scope.Skip(10),

		// Page
		scope.Page(1, 20),
	).Find(&[]struct{}{})
}
```