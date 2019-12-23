/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/transport/compress"
)

// WebSocketConn represents a websocket connection interface.
type WebSocketConn interface {
	NextReader() (messageType int, r io.Reader, err error)
	NextWriter(int) (io.WriteCloser, error)
	Close() error
	UnderlyingConn() net.Conn
	SetReadDeadline(t time.Time) error
}

type webSocketTransport struct {
	conn      WebSocketConn
	keepAlive time.Duration
}

// NewWebSocketTransport creates a socket class stream transport.
func NewWebSocketTransport(conn WebSocketConn, keepAlive time.Duration) Transport {
	wst := &webSocketTransport{
		conn:      conn,
		keepAlive: keepAlive,
	}
	return wst
}

func (w *webSocketTransport) Read(p []byte) (n int, err error) {
	_, r, err := w.conn.NextReader()
	if err != nil {
		return 0, err
	}
	if w.keepAlive > 0 {
		_ = w.conn.SetReadDeadline(time.Now().Add(w.keepAlive))
	}
	return r.Read(p)
}

func (w *webSocketTransport) Write(p []byte) (n int, err error) {
	nw, err := w.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	defer func() { _ = nw.Close() }()

	return nw.Write(p)
}

func (w *webSocketTransport) Close() error {
	return w.conn.Close()
}

func (w *webSocketTransport) Type() Type {
	return WebSocket
}

func (w *webSocketTransport) WriteString(str string) (int, error) {
	nw, err := w.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	defer func() { _ = nw.Close() }()

	n, err := io.Copy(nw, strings.NewReader(str))
	return int(n), err
}

// Flush writes any buffered data to the underlying io.Writer.
func (w *webSocketTransport) Flush() error {
	return nil
}

// SetWriteDeadline sets the deadline for future write calls.
func (w *webSocketTransport) SetWriteDeadline(d time.Time) error {
	return w.conn.UnderlyingConn().SetWriteDeadline(d)
}

func (w *webSocketTransport) StartTLS(_ *tls.Config, _ bool) {
}

func (w *webSocketTransport) EnableCompression(_ compress.Level) {
}

func (w *webSocketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
	if tlsConn, ok := w.conn.UnderlyingConn().(tlsStateQueryable); ok {
		switch mechanism {
		case TLSUnique:
			st := tlsConn.ConnectionState()
			return st.TLSUnique
		default:
			break
		}
	}
	return nil
}

func (w *webSocketTransport) PeerCertificates() []*x509.Certificate {
	if tlsConn, ok := w.conn.UnderlyingConn().(tlsStateQueryable); ok {
		st := tlsConn.ConnectionState()
		return st.PeerCertificates
	}
	return nil
}
