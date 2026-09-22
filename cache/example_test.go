package cache_test

import (
	"context"
	"time"

	"github.com/go-fries/fries/cache/v4"
)

func ExampleTyped() {
	type User struct{ Name string }

	// Supply the application's cache backend and database lookup at startup.
	newUserQuery := func(store cache.ReadWriter, findUser func(context.Context, string) (User, error)) func(context.Context, string) (User, error) {
		users := cache.Typed[User](store)
		return func(ctx context.Context, id string) (User, error) {
			return users.Remember(ctx, "users:"+id, 5*time.Minute, func(ctx context.Context) (User, error) {
				return findUser(ctx, id)
			})
		}
	}
	_ = newUserQuery
}
