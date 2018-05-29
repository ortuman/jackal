/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/transport/compress"
	"github.com/ortuman/jackal/xml"
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
	keepAlive int
}

// NewWebSocketTransport creates a socket class stream transport.
func NewWebSocketTransport(conn WebSocketConn, keepAlive int) Transport {
	wst := &webSocketTransport{
		conn:      conn,
		keepAlive: keepAlive,
	}
	return wst
}

func (wst *webSocketTransport) Type() TransportType {
	return WebSocket
}

func (wst *webSocketTransport) Read(p []byte) (n int, err error) {
	_, r, err := wst.conn.NextReader()
	if err != nil {
		return 0, err
	}
	wst.conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(wst.keepAlive)))
	return r.Read(p)
}

func (wst *webSocketTransport) WriteString(str string) error {
	w, err := wst.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.WriteString(w, str)
	return err
}

func (wst *webSocketTransport) WriteElement(elem xml.XElement, includeClosing bool) error {
	w, err := wst.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	defer w.Close()
	elem.ToXML(w, includeClosing)
	return nil
}

func (wst *webSocketTransport) Close() error {
	return wst.conn.Close()
}

func (wst *webSocketTransport) StartTLS(cfg *tls.Config) {
}

func (wst *webSocketTransport) EnableCompression(level compress.Level) {
}

func (wst *webSocketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
	if tlsConn, ok := wst.conn.UnderlyingConn().(*tls.Conn); ok {
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
