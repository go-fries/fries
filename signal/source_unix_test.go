//go:build unix

package signal

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	ossignal "os/signal"
	"regexp"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const signalTestChildEnv = "FRIES_SIGNAL_TEST_CHILD"

// Deliver real signals only to a dedicated child, never to the Go test process.
func TestRouterOSSignalProcess(t *testing.T) {
	if os.Getenv(signalTestChildEnv) == t.Name() {
		runSignalChild(t)
		return
	}

	runSignalProcess(t, func(scanner *bufio.Scanner, process *os.Process) {
		require.True(t, scanner.Scan(), "child did not install subscriptions")
		require.Equal(t, "ready", scanner.Text())
		require.NoError(t, process.Signal(syscall.SIGUSR1))
		require.True(t, scanner.Scan(), "child did not handle the signal")
		assert.Equal(t, "handled", scanner.Text())
		require.True(t, scanner.Scan(), "child did not verify subscriber isolation")
		assert.Equal(t, "foreign subscription retained", scanner.Text())
	})
}

func TestOSSignalSourceProcess(t *testing.T) {
	if os.Getenv(signalTestChildEnv) == t.Name() {
		runSourceChild(t)
		return
	}

	runSignalProcess(t, func(scanner *bufio.Scanner, _ *os.Process) {
		require.True(t, scanner.Scan(), "child did not verify OS signal source")
		assert.Equal(t, "source verified", scanner.Text())
	})
}

func runSignalProcess(t *testing.T, check func(*bufio.Scanner, *os.Process)) {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	args := []string{"-test.run=^" + regexp.QuoteMeta(t.Name()) + "$"}
	// Share Go's per-run coverage directory, not its final profile. Otherwise
	// wrapper calls made only in the child are absent from the parent's report.
	if coverageDir := flag.Lookup("test.gocoverdir"); coverageDir != nil && coverageDir.Value.String() != "" {
		args = append(args, "-test.gocoverdir="+coverageDir.Value.String())
	}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = append(os.Environ(), signalTestChildEnv+"="+t.Name())
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
	check(scanner, cmd.Process)
	// Drain Go's final PASS and coverage output before waiting for the process.
	for scanner.Scan() {
	}
	require.NoError(t, scanner.Err())
	err = cmd.Wait()
	waited = true
	require.NoError(t, err, "child stderr: %s", stderr.String())
}

func runSourceChild(t *testing.T) {
	process, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	source := osSignalSource{}

	t.Run("notify forwards multiple signals", func(t *testing.T) {
		incoming := make(chan os.Signal, 2)
		source.Notify(incoming, syscall.SIGUSR1, syscall.SIGUSR2)
		t.Cleanup(func() { source.Stop(incoming) })

		for _, sig := range []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2} {
			require.NoError(t, process.Signal(sig))
			assert.Equal(t, sig, receiveOSSignal(t, incoming))
		}
	})

	t.Run("stop detaches only its channel and leaves it open", func(t *testing.T) {
		incoming := make(chan os.Signal, 1)
		foreign := make(chan os.Signal, 1)
		source.Notify(incoming, syscall.SIGUSR1)
		t.Cleanup(func() { source.Stop(incoming) })
		ossignal.Notify(foreign, syscall.SIGUSR1)
		t.Cleanup(func() { ossignal.Stop(foreign) })

		source.Stop(incoming)
		source.Stop(incoming) // Repeated Stop must remain safe.
		require.NoError(t, process.Signal(syscall.SIGUSR1))
		assert.Equal(t, syscall.SIGUSR1, receiveOSSignal(t, foreign))
		require.Empty(t, incoming)

		// Stop must neither close the channel nor prevent its owner from using it.
		require.NotPanics(t, func() { incoming <- syscall.SIGUSR2 })
		assert.Equal(t, syscall.SIGUSR2, <-incoming)
		require.NotPanics(t, func() { close(incoming) })
	})

	t.Run("stop unregistered channel", func(t *testing.T) {
		incoming := make(chan os.Signal, 1)
		require.NotPanics(t, func() { source.Stop(incoming) })
		require.NotPanics(t, func() { incoming <- syscall.SIGUSR1 })
		assert.Equal(t, syscall.SIGUSR1, <-incoming)
	})

	if !t.Failed() {
		fmt.Println("source verified")
	}
}

func receiveOSSignal(t *testing.T, incoming <-chan os.Signal) os.Signal {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case sig := <-incoming:
		return sig
	case <-ctx.Done():
		t.Fatal("timed out waiting for OS signal")
		return nil
	}
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
	assert.Equal(t, syscall.SIGUSR1, receiveOSSignal(t, foreign))
	process, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, process.Signal(syscall.SIGUSR1))
	assert.Equal(t, syscall.SIGUSR1, receiveOSSignal(t, foreign))
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
