package signal_test

import (
	"context"
	"log/slog"
	"os"
	ossignal "os/signal"
	"syscall"
	"time"

	"github.com/go-fries/fries/signal/v4"
)

func ExampleRouter_Listen() {
	router := signal.New()
	if err := router.Listen(syscall.SIGHUP, func(ctx context.Context, _ os.Signal) error {
		return reload(ctx)
	}); err != nil {
		panic(err)
	}
	// Run and shut down router using the lifecycle shown in ExampleRouter_Run.
}

// This example compiles the lifecycle wiring. Run it as a Unix application and
// send SIGHUP to trigger reload; send SIGINT or SIGTERM to request shutdown.
func ExampleRouter_Run() {
	router := signal.New()
	if err := router.Listen(syscall.SIGHUP, func(ctx context.Context, _ os.Signal) error {
		return reload(ctx)
	}); err != nil {
		panic(err)
	}
	exitCtx, stopExit := ossignal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopExit()
	// Keep the Run context separate so an exit signal does not cancel active
	// handlers before graceful shutdown waits for them.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- router.Run(ctx) }()

	<-exitCtx.Done()
	stopExit()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := router.Shutdown(shutdownCtx); err != nil {
		slog.Error("signal shutdown failed", "error", err)
	}
	// Joining is still necessary. A handler that ignores cancellation can
	// prevent completion even after Shutdown's timeout.
	if err := <-done; err != nil {
		slog.Error("signal router failed", "error", err)
	}
}

func ExampleRouter_Register() {
	router := signal.New()
	handler := &reloadHandler{reload: reload}
	if err := router.Register(handler, syscall.SIGHUP, syscall.SIGUSR1); err != nil {
		panic(err)
	}
	// Run and shut down router using the lifecycle shown in ExampleRouter_Run.
}

type reloadHandler struct {
	reload func(context.Context) error
}

func (h *reloadHandler) Handle(ctx context.Context, _ os.Signal) error {
	return h.reload(ctx)
}

func reload(ctx context.Context) error {
	return ctx.Err() // Replace with a context-aware configuration reload.
}
