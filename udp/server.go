package udp

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/go-kratos/kratos/v2/transport"
)

type Message struct {
	Conn net.PacketConn
	Addr net.Addr
	Body []byte
}

type Server struct {
	address string

	bufSize int

	conn net.PacketConn

	handler func(message *Message)

	recoveryHandler func(message *Message, err any)

	readChan     chan *Message
	readChanSize int // readChan size

	stoped     chan struct{}
	stopedOnce sync.Once
}

var _ transport.Server = (*Server)(nil)

type Option func(*Server)

func WithBufSize(bufSize int) Option {
	return func(s *Server) {
		if bufSize > 0 {
			s.bufSize = bufSize
		}
	}
}

func WithHandler(handler func(message *Message)) Option {
	return func(s *Server) {
		if handler != nil {
			s.handler = handler
		}
	}
}

func WithRecoveryHandler(handler func(message *Message, err any)) Option {
	return func(s *Server) {
		if handler != nil {
			s.recoveryHandler = handler
		}
	}
}

func WithReadChanSize(readChanSize int) Option {
	return func(s *Server) {
		if readChanSize > 0 {
			s.readChanSize = readChanSize
		}
	}
}

func NewServer(address string, opts ...Option) *Server {
	s := &Server{
		address:      address,
		bufSize:      1024, //nolint:mnd
		readChanSize: 1024, //nolint:mnd
		stoped:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.readChan = make(chan *Message, s.readChanSize)

	return s
}

func (s *Server) Start(_ context.Context) (err error) {
	s.conn, err = net.ListenPacket("udp", s.address)
	if err != nil {
		return
	}

	log.Printf("udp server: listening on %s\n", s.address)

	go s.start()

	buf := make([]byte, s.bufSize)

	for {
		n, addr, err := s.conn.ReadFrom(buf)
		if err != nil {
			s.stop()
			return err
		}

		s.readChan <- &Message{
			Conn: s.conn,
			Addr: addr,
			Body: buf[:n],
		}
	}
}

func (s *Server) start() {
	for {
		select {
		case <-s.stoped:
			return
		case message := <-s.readChan:
			if s.handler != nil {
				s.handle(message)
			}
		}
	}
}

func (s *Server) handle(message *Message) {
	if s.recoveryHandler != nil {
		defer func() {
			if err := recover(); err != nil {
				s.recoveryHandler(message, err)
			}
		}()
	}

	s.handler(message)
}

func (s *Server) Stop(_ context.Context) error {
	log.Println("udp server: stopping")

	s.stop()

	if s.conn == nil {
		return nil
	}

	return s.conn.Close()
}

func (s *Server) stop() {
	s.stopedOnce.Do(func() {
		close(s.stoped)
	})
}
