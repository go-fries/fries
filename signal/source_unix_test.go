//go:build unix

package signal

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	ossignal "os/signal"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Deliver real signals only to a dedicated child, never to the Go test process.
func TestRouterOSSignalProcess(t *testing.T) {
	if os.Getenv("FRIES_SIGNAL_ROUTER_CHILD") == "1" {
		runSignalChild(t)
		return
	}

	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestRouterOSSignalProcess$")
	cmd.Env = append(os.Environ(), "FRIES_SIGNAL_ROUTER_CHILD=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	var waited bool
	t.Cleanup(func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	scanner := bufio.NewScanner(stdout)
	require.True(t, scanner.Scan(), "child did not install subscriptions")
	require.Equal(t, "ready", scanner.Text())
	require.NoError(t, cmd.Process.Signal(syscall.SIGUSR1))
	require.True(t, scanner.Scan(), "child did not handle the signal")
	assert.Equal(t, "handled", scanner.Text())
	require.True(t, scanner.Scan(), "child did not verify subscriber isolation")
	assert.Equal(t, "foreign subscription retained", scanner.Text())
	// Drain Go's final PASS output before waiting for the process.
	for scanner.Scan() {
	}
	require.NoError(t, scanner.Err())
	err = cmd.Wait()
	waited = true
	require.NoError(t, err, "child stderr: %s", stderr.String())
}

func runSignalChild(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	foreign := make(chan os.Signal, 2)
	ossignal.Notify(foreign, syscall.SIGUSR1)
	defer ossignal.Stop(foreign)

	r := quietRouter()
	r.source = readyOSSource{}
	var calls atomic.Int32
	require.NoError(t, r.Listen(syscall.SIGUSR1, func(context.Context, os.Signal) error {
		calls.Add(1)
		fmt.Println("handled")
		cancel()
		return nil
	}))
	if err := r.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned %v, want context cancellation", err)
	}
	// Consume the first copy, then prove Router cleanup did not reset this
	// independent subscriber by delivering a second signal after Run joined.
	<-foreign
	process, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, process.Signal(syscall.SIGUSR1))
	assert.Equal(t, syscall.SIGUSR1, <-foreign)
	assert.Equal(t, int32(1), calls.Load())
	fmt.Println("foreign subscription retained")
}

type readyOSSource struct {
	osSignalSource
}

func (readyOSSource) Notify(ch chan<- os.Signal, signals ...os.Signal) {
	osSignalSource{}.Notify(ch, signals...)
	fmt.Println("ready")
}
