/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
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
	r         *bytes.Reader
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

func (wst *webSocketTransport) Read(p []byte) (n int, err error) {
	_, r, err := wst.conn.NextReader()
	if err != nil {
		return 0, err
	}
	if wst.keepAlive > 0 {
		wst.conn.SetReadDeadline(time.Now().Add(wst.keepAlive))
	}
	return r.Read(p)
}

func (wst *webSocketTransport) Write(p []byte) (n int, err error) {
	w, err := wst.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	defer w.Close()
	return w.Write(p)
}

func (wst *webSocketTransport) Close() error {
	return wst.conn.Close()
}

func (wst *webSocketTransport) Type() TransportType {
	return WebSocket
}

func (wst *webSocketTransport) WriteString(str string) (int, error) {
	w, err := wst.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	defer w.Close()
	n, err := io.Copy(w, strings.NewReader(str))
	return int(n), err
}

func (wst *webSocketTransport) StartTLS(_ *tls.Config, _ bool) {
}

func (wst *webSocketTransport) EnableCompression(level compress.Level) {
}

func (wst *webSocketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
	if tlsConn, ok := wst.conn.UnderlyingConn().(tlsStateQueryable); ok {
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

func (wst *webSocketTransport) PeerCertificates() []*x509.Certificate {
	if tlsConn, ok := wst.conn.UnderlyingConn().(tlsStateQueryable); ok {
		st := tlsConn.ConnectionState()
		return st.PeerCertificates
	}
	return nil
}
