# Gorm/Scopes

## Tests

Run commands from the repository root:

```sh
# Compile the examples and skip MySQL integration tests.
make test/gorm/scope ARGS='-short -race -count=1'

# Run integration tests against an existing MySQL database.
MYSQL_DSN='gorm:gorm@tcp(localhost:3306)/gorm?charset=utf8&parseTime=True&loc=Local' \
  make test/gorm/scope ARGS='-race -count=1'
```

The DSN above is the default when `MYSQL_DSN` is unset. Outside short mode, connection or setup failures fail the test. The helper enables time parsing and defaults unset connection, read, and write timeouts to three seconds.

Each test creates its own randomly prefixed `users` table in the configured database and drops that table before closing its SQL connection pool. The database must already exist; the account needs table creation, index creation, querying, insertion, and table deletion privileges within it. Database creation privileges are not required. Existing tables and rows are preserved, so independent test processes can share the same database.

## Example

```go
package scope_test

import (
	"time"

	"github.com/go-fries/fries/gorm/scope/v4"
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
