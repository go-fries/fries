package udp

import (
	"bytes"
	"context"
	"log"
	"net"
	"sync"

	"github.com/go-kratos/kratos/v3/transport"
)

type Message struct {
	Conn net.PacketConn
	Addr net.Addr
	Body []byte
}

func newMessage(conn net.PacketConn, addr net.Addr, body []byte) *Message {
	return &Message{
		Conn: conn,
		Addr: addr,
		Body: bytes.Clone(body),
	}
}

type Server struct {
	address string

	bufSize int

	connMu sync.Mutex
	conn   net.PacketConn

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
	conn, err := net.ListenPacket("udp", s.address)
	if err != nil {
		return err
	}
	s.connMu.Lock()
	select {
	case <-s.stoped:
		s.connMu.Unlock()
		_ = conn.Close()
		return net.ErrClosed
	default:
		s.conn = conn
		s.connMu.Unlock()
	}
	defer func() {
		s.connMu.Lock()
		if s.conn == conn {
			s.conn = nil
		}
		s.connMu.Unlock()
		_ = conn.Close()
	}()

	log.Printf("udp server: listening on %s\n", s.address)

	go s.start()
	return s.serve(conn)
}

func (s *Server) serve(conn net.PacketConn) error {
	buf := make([]byte, s.bufSize)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			s.stop()
			return err
		}

		if err := s.enqueue(newMessage(conn, addr, buf[:n])); err != nil {
			return err
		}
	}
}

func (s *Server) enqueue(message *Message) error {
	select {
	case s.readChan <- message:
		return nil
	case <-s.stoped:
		return net.ErrClosed
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

	s.connMu.Lock()
	conn := s.conn
	s.conn = nil
	s.connMu.Unlock()
	if conn == nil {
		return nil
	}

	return conn.Close()
}

func (s *Server) stop() {
	s.stopedOnce.Do(func() {
		close(s.stoped)
	})
}
