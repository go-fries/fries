package signal

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/go-fries/fries/contract/v3"
	"github.com/go-kratos/kratos/v2/log"
)

var DefaultRecovery = func(err any, sig os.Signal, _ Handler) {
	log.Errorf("[Signal] handler panic (%s): %v", sig, err)
}

type Server struct {
	handlers []Handler
	stopped  chan struct{}
	stopOnce sync.Once
	mu       sync.RWMutex
	recovery func(any, os.Signal, Handler)
}

type Option func(*Server)

func AddHandler(handler ...Handler) Option {
	return func(s *Server) {
		s.handlers = append(s.handlers, handler...)
	}
}

func WithRecovery(handler func(any, os.Signal, Handler)) Option {
	return func(s *Server) {
		if handler != nil {
			s.recovery = handler
		}
	}
}

func NewServer(opts ...Option) *Server {
	server := &Server{
		handlers: make([]Handler, 0),
		stopped:  make(chan struct{}),
	}

	for _, opt := range opts {
		opt(server)
	}

	return server
}

func (s *Server) Start(ctx context.Context) error {
	handlers, signals := buildHandlers(s.snapshotHandlers())

	log.Infof("[Signal] server starting")

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
					if async, ok := h.(contract.Asyncable); ok && async.Async() {
						go s.handle(sig, h)
					} else {
						s.handle(sig, h)
					}
				}
			}
		}
	}
}

func (s *Server) Register(handlers ...Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers = append(s.handlers, handlers...)
}

func (s *Server) Stop(context.Context) error {
	s.stopOnce.Do(func() {
		log.Infof("[Signal] server stopping")
		close(s.stopped)
	})
	return nil
}

func (s *Server) handle(sig os.Signal, handler Handler) {
	defer func() {
		if s.recovery != nil {
			if err := recover(); err != nil {
				s.recovery(err, sig, handler)
			}
		}
	}()

	handler.Handle(sig)
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
