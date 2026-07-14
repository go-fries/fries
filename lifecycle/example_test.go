package lifecycle_test

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/lifecycle/v4"
)

func ExampleRunner_Run() {
	provider := lifecycle.BootstrapFunc(
		func(ctx context.Context) (context.Context, error) {
			fmt.Println("bootstrap")
			return ctx, nil
		},
	)

	runner := lifecycle.New(lifecycle.WithProviders(provider))
	_ = runner.Run(context.Background(), func(context.Context) error {
		fmt.Println("run")
		return nil
	})

	// Output:
	// bootstrap
	// run
}
