package gate

import (
	"cmp"
	"compress/zlib"
	"context"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rc4"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mmm-revived/internal/wire"
)

const (
	DefaultHandshakeTimeout = 15 * time.Second
	DefaultIdleTimeout      = 2 * time.Minute
	DefaultWriteTimeout     = 30 * time.Second
	DefaultSendQueue        = 1024
)

const (
	methodSeed uint16 = iota
	methodSessionKey
	methodConnect
	methodEntity
)

var (
	errExpectedSessionKey = errors.New("expected session key")
	errExpectedConnect    = errors.New("expected connect request")
)

type Handler interface {
	Open(*Session) (Connection, error)
}

type Connection interface {
	Receive(EntityMessage) error
	Close()
}

type Server struct {
	Key              *rsa.PrivateKey
	Handler          Handler
	Logger           *slog.Logger
	HandshakeTimeout time.Duration
	IdleTimeout      time.Duration
	WriteTimeout     time.Duration
	SendQueue        int
}

func LoadKey(privatePath, publicPath string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(privatePath)
	if err == nil {
		block, _ := pem.Decode(data)
		if block == nil || block.Type != "RSA PRIVATE KEY" {
			return nil, errors.New("invalid server key")
		}
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if err = validatePublicKey(publicPath, key); err != nil {
			return nil, err
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if _, err := os.Stat(publicPath); err == nil {
		return nil, errors.New("server public key exists without its private key")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	for _, file := range []struct {
		path  string
		block *pem.Block
		mode  os.FileMode
	}{
		{privatePath, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}, 0600},
		{publicPath, &pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey)}, 0644},
	} {
		if err = os.MkdirAll(filepath.Dir(file.path), 0700); err != nil {
			return nil, err
		}
		if err = createKeyFile(file.path, pem.EncodeToMemory(file.block), file.mode); err != nil {
			return nil, err
		}
	}
	return key, nil
}

func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	stop := context.AfterFunc(ctx, func() { listener.Close() })
	defer stop()
	var sessions sync.WaitGroup
	defer sessions.Wait()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		sessions.Go(func() {
			defer conn.Close()
			stop := context.AfterFunc(ctx, func() { conn.Close() })
			defer stop()
			session := &Session{Conn: conn, Logger: s.Logger.With("peer", conn.RemoteAddr().String()), reader: conn, writer: conn}
			if err := s.serveSession(session); err != nil && !errors.Is(err, io.EOF) && ctx.Err() == nil {
				session.Logger.Warn("connection closed", "error", err)
			}
		})
	}
}

func (s *Server) serveSession(session *Session) error {
	if err := s.handshake(session); err != nil {
		return err
	}
	if err := session.Conn.SetDeadline(time.Time{}); err != nil {
		return err
	}
	session.start(cmp.Or(max(s.SendQueue, 0), DefaultSendQueue), cmp.Or(max(s.WriteTimeout, 0), DefaultWriteTimeout))
	defer session.stop()
	connection, err := s.Handler.Open(session)
	if err != nil {
		return err
	}
	defer connection.Close()
	zReader, err := zlib.NewReader(session.reader)
	if err != nil {
		return err
	}
	defer zReader.Close()
	session.reader = zReader
	idle := cmp.Or(max(s.IdleTimeout, 0), DefaultIdleTimeout)
	for {
		if err = session.Conn.SetReadDeadline(time.Now().Add(idle)); err != nil {
			return err
		}
		fields, err := session.expect(methodEntity, nil)
		if err != nil {
			return err
		}
		message := EntityMessage{Entity: wire.FindField(fields, 1).Data, Payload: wire.FindField(fields, 2).Data, Channel: wire.FindField(fields, 3).Data}
		if err = connection.Receive(message); err != nil {
			return err
		}
	}
}

func (s *Server) handshake(session *Session) error {
	if err := session.Conn.SetDeadline(time.Now().Add(cmp.Or(max(s.HandshakeTimeout, 0), DefaultHandshakeTimeout))); err != nil {
		return err
	}
	p, err := wire.ReadPacket(session.reader)
	if err != nil {
		return err
	}
	if p.Method != methodSeed || len(p.Data) != 0 {
		return errors.New("expected seed request")
	}
	var random [8]byte
	rand.Read(random[:])
	seed := binary.LittleEndian.Uint64(random[:]) & 0x7fffffffffffffff
	if err = session.send(methodSeed, wire.UintField(nil, 1, seed)); err != nil {
		return err
	}
	fields, err := session.expect(methodSessionKey, errExpectedSessionKey)
	if err != nil {
		return err
	}
	plain, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, s.Key, wire.FindField(fields, 1).Data, nil)
	if err != nil {
		return fmt.Errorf("decrypt session key: %w; check that the client installed this server's public-key patch", err)
	}
	if fields, err = wire.ParseFields(plain); err != nil {
		return err
	}
	if wire.FindField(fields, 3).Value != seed {
		return errors.New("session seed mismatch")
	}
	key := wire.FindField(fields, 2).Data
	if len(key) < 16 || len(key) > 256 {
		return errors.New("invalid session key length")
	}
	readCipher, err := rc4.NewCipher(key)
	if err != nil {
		return err
	}
	writeCipher, err := rc4.NewCipher(key)
	if err != nil {
		return err
	}
	session.reader = &cipher.StreamReader{S: readCipher, R: session.Conn}
	session.writer = &cipher.StreamWriter{S: writeCipher, W: session.Conn}
	if err = session.send(methodSessionKey, nil); err != nil {
		return err
	}
	if fields, err = session.expect(methodConnect, errExpectedConnect); err != nil {
		return err
	}
	session.Logger.Info("native handshake accepted", "device", string(wire.FindField(fields, 2).Data))
	session.zWriter = zlib.NewWriter(session.writer)
	session.writer = session.zWriter
	return session.send(methodConnect, wire.UintField(nil, 2, 1))
}
