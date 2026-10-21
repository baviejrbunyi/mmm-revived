package bootstrap

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"mmm-revived/internal/json"
)

const (
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultReadTimeout       = 10 * time.Second
	DefaultWriteTimeout      = 3 * time.Minute
	DefaultIdleTimeout       = 30 * time.Second
	DefaultShutdownTimeout   = 10 * time.Second
)

type Server struct {
	GateAddress       string
	NativePatches     http.Handler
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type Health struct {
	Status string `json:"status"`
}

func (s Server) handler() (http.Handler, error) {
	host, portText, err := net.SplitHostPort(s.GateAddress)
	if err != nil {
		return nil, err
	}
	if net.ParseIP(host) == nil {
		return nil, errors.New("advertised gate host must be an IP address")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return nil, errors.New("invalid advertised gate port")
	}
	serverList := fmt.Appendf(nil, "Test 0 3 0 Local Local %s:%d:1:0 0 0 1 cn\n", host, port)
	health, err := json.Marshal(Health{Status: "listening"})
	if err != nil {
		return nil, err
	}
	health = append(health, '\n')
	router := gin.New()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	router.HandleMethodNotAllowed = true
	router.ForwardedByClientIP = false
	if err = router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	router.Use(gin.Recovery())
	router.NoRoute(func(c *gin.Context) {
		http.NotFound(c.Writer, c.Request)
	})
	router.NoMethod(func(c *gin.Context) {
		if s.NativePatches != nil && strings.HasPrefix(c.Request.URL.Path, "/native/") {
			s.NativePatches.ServeHTTP(c.Writer, c.Request)
			return
		}
		http.Error(c.Writer, "method not allowed", http.StatusMethodNotAllowed)
	})
	if s.NativePatches != nil {
		router.Match([]string{http.MethodGet, http.MethodHead}, "/native/*path", gin.WrapH(s.NativePatches))
	}
	router.Match([]string{http.MethodGet, http.MethodHead}, "/serverlist", cachedResponse("text/plain; charset=utf-8", serverList))
	router.Match([]string{http.MethodGet, http.MethodHead}, "/health", cachedResponse("application/json", health))
	return router, nil
}

func cachedResponse(contentType string, data []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", contentType)
		c.Header("Content-Length", strconv.Itoa(len(data)))
		if c.Request.Method == http.MethodHead {
			c.Status(http.StatusOK)
			return
		}
		c.Data(http.StatusOK, contentType, data)
	}
}

func (s Server) Serve(ctx context.Context, listener net.Listener) error {
	defer listener.Close()
	handler, err := s.handler()
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: cmp.Or(max(s.ReadHeaderTimeout, 0), DefaultReadHeaderTimeout),
		ReadTimeout:       cmp.Or(max(s.ReadTimeout, 0), DefaultReadTimeout),
		WriteTimeout:      cmp.Or(max(s.WriteTimeout, 0), DefaultWriteTimeout),
		IdleTimeout:       cmp.Or(max(s.IdleTimeout, 0), DefaultIdleTimeout),
	}
	stopped := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(stopped)
		shutdown, cancel := context.WithTimeout(context.Background(), cmp.Or(max(s.ShutdownTimeout, 0), DefaultShutdownTimeout))
		defer cancel()
		if server.Shutdown(shutdown) != nil {
			server.Close()
		}
	})
	err = server.Serve(listener)
	if !errors.Is(err, http.ErrServerClosed) {
		stop()
		return err
	}
	<-stopped
	return nil
}
