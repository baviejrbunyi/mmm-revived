package gate

import (
	"cmp"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"mmm-revived/internal/wire"
)

var (
	ErrSendQueueFull = errors.New("client send queue is full")
	ErrSessionClosed = errors.New("client session is closed")
)

type Session struct {
	Conn      net.Conn
	Logger    *slog.Logger
	Entity    []byte
	Channel   []byte
	reader    io.Reader
	writer    io.Writer
	zWriter   *zlib.Writer
	mu        sync.Mutex
	queue     chan []byte
	closed    bool
	done      chan struct{}
	closeOnce sync.Once
}

type EntityMessage struct {
	Entity  []byte
	Channel []byte
	Payload []byte
}

func (s *Session) Send(payload []byte) error {
	data := make([]byte, 0, len(s.Entity)+len(payload)+len(s.Channel)+24)
	data = wire.BytesField(data, 1, s.Entity)
	data = wire.BytesField(data, 2, payload)
	data = wire.BytesField(data, 3, s.Channel)
	data = wire.UintField(data, 4, 0)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrSessionClosed
	}
	select {
	case s.queue <- data:
		return nil
	default:
		s.closed = true
		close(s.queue)
		s.Conn.Close()
		return ErrSendQueueFull
	}
}

func (s *Session) CloseAfterFlush(timeout time.Duration) {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		if !s.closed {
			s.closed = true
			if s.queue != nil {
				close(s.queue)
			}
		}
		s.mu.Unlock()
		go func() {
			timer := time.NewTimer(cmp.Or(max(timeout, 0), time.Second))
			defer timer.Stop()
			select {
			case <-s.done:
			case <-timer.C:
			}
			s.Conn.Close()
		}()
	})
}

func (s *Session) expect(method uint16, mismatch error) ([]wire.Field, error) {
	p, err := wire.ReadPacket(s.reader)
	if err != nil {
		return nil, err
	}
	if p.Method != method {
		if mismatch == nil {
			return nil, fmt.Errorf("unsupported gate method %d", p.Method)
		}
		return nil, mismatch
	}
	return wire.ParseFields(p.Data)
}

func (s *Session) send(method uint16, data []byte) error {
	if err := wire.WritePacket(s.writer, wire.Packet{Method: method, Data: data}); err != nil {
		return err
	}
	if s.zWriter != nil {
		return s.zWriter.Flush()
	}
	return nil
}

func (s *Session) start(capacity int, timeout time.Duration) {
	s.queue = make(chan []byte, capacity)
	s.done = make(chan struct{})
	go s.drain(timeout)
}

func (s *Session) stop() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.queue)
	}
	s.mu.Unlock()
	<-s.done
}

func (s *Session) drain(timeout time.Duration) {
	defer close(s.done)
	for frame := range s.queue {
		if err := s.burst(frame, timeout); err != nil {
			s.mu.Lock()
			s.closed = true
			s.mu.Unlock()
			s.Conn.Close()
			s.Logger.Debug("client write failed", "error", err)
			return
		}
	}
}

func (s *Session) burst(frame []byte, timeout time.Duration) error {
	if err := s.Conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	for {
		if err := wire.WriteBufferedPacket(s.writer, wire.Packet{Method: methodEntity, Data: frame}); err != nil {
			return err
		}
		select {
		case next, ok := <-s.queue:
			if !ok {
				return s.zWriter.Flush()
			}
			frame = next
		default:
			return s.zWriter.Flush()
		}
	}
}
