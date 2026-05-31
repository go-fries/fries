package signal

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

// DefaultRecovery logs panics raised by signal handlers.
var DefaultRecovery RecoveryHandler = func(ctx context.Context, sig os.Signal, _ Handler, recovered any) {
	log.Context(ctx).Errorf("[Signal] handler panic (%s): %v", sig, recovered)
}

// Server routes operating system signals to registered handlers.
type Server struct {
	handlers []Handler
	stopped  chan struct{}
	stopOnce sync.Once
	mu       sync.RWMutex
	recovery RecoveryHandler
}

// NewServer creates a Server with the supplied options.
func NewServer(opts ...Option) *Server {
	cfg := config{
		handlers: make([]Handler, 0),
	}

	for _, opt := range opts {
		opt.apply(&cfg)
	}

	server := &Server{
		handlers: cfg.handlers,
		stopped:  make(chan struct{}),
		recovery: cfg.recovery,
	}

	return server
}

// Start subscribes to registered signals and blocks until the context is done or the server stops.
func (s *Server) Start(ctx context.Context) error {
	handlers, signals := buildHandlers(s.snapshotHandlers())

	log.Context(ctx).Infof("[Signal] server starting")

	if len(signals) == 0 {
		return s.wait(ctx)
	}

	ch := make(chan os.Signal, len(signals))
	signal.Notify(ch, signals...)
	defer signal.Stop(ch)

	return s.serve(ctx, ch, handlers)
}

func (s *Server) serve(ctx context.Context, ch <-chan os.Signal, handlers map[os.Signal][]Handler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopped:
			return nil
		case sig := <-ch:
			if hs, ok := handlers[sig]; ok {
				for _, h := range hs {
					if _, ok := h.(AsyncHandler); ok {
						go s.handle(ctx, sig, h)
					} else {
						s.handle(ctx, sig, h)
					}
				}
			}
		}
	}
}

// Register adds handlers to the Server.
func (s *Server) Register(handlers ...Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers = append(s.handlers, handlers...)
}

// Stop stops the Server and unblocks Start.
func (s *Server) Stop(ctx context.Context) error {
	s.stopOnce.Do(func() {
		log.Context(ctx).Infof("[Signal] server stopping")
		close(s.stopped)
	})
	return nil
}

func (s *Server) handle(ctx context.Context, sig os.Signal, handler Handler) {
	defer func() {
		if s.recovery != nil {
			if err := recover(); err != nil {
				s.recovery(ctx, sig, handler, err)
			}
		}
	}()

	handler.Handle(ctx, sig)
}

func (s *Server) wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.stopped:
		return nil
	}
}

func (s *Server) snapshotHandlers() []Handler {
	s.mu.RLock()
	defer s.mu.RUnlock()

	handlers := make([]Handler, len(s.handlers))
	copy(handlers, s.handlers)
	return handlers
}

func buildHandlers(handlers []Handler) (map[os.Signal][]Handler, []os.Signal) {
	routed := make(map[os.Signal][]Handler)
	signals := make([]os.Signal, 0)
	seen := make(map[os.Signal]struct{})

	for _, h := range handlers {
		handlerSeen := make(map[os.Signal]struct{})
		for _, sig := range h.Listen() {
			if _, ok := handlerSeen[sig]; ok {
				continue
			}
			handlerSeen[sig] = struct{}{}
			routed[sig] = append(routed[sig], h)
			if _, ok := seen[sig]; !ok {
				seen[sig] = struct{}{}
				signals = append(signals, sig)
			}
		}
	}

	return routed, signals
}
