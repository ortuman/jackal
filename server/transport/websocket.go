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
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/server/compress"
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

type websocketTransport struct {
	conn          WebSocketConn
	r             *bytes.Reader
	rbuf          []byte
	p             *xml.Parser
	maxStanzaSize int
	keepAlive     int
}

// NewSocketTransport creates a socket class stream transport.
func NewWebSocketTransport(conn WebSocketConn, maxStanzaSize, keepAlive int) Transport {
	wst := &websocketTransport{
		conn:          conn,
		rbuf:          make([]byte, maxStanzaSize+1),
		maxStanzaSize: maxStanzaSize,
		keepAlive:     keepAlive,
	}
	return wst
}

func (wst *websocketTransport) ReadElement() (xml.XElement, error) {
	if err := wst.readFromConn(); err != nil {
		return nil, err
	}
	return wst.p.ParseElement()
}

func (wst *websocketTransport) WriteString(str string) error {
	w, err := wst.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, strings.NewReader(str))
	return err
}

func (wst *websocketTransport) WriteElement(elem xml.XElement, includeClosing bool) error {
	w, err := wst.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	defer w.Close()
	elem.ToXML(w, includeClosing)
	return nil
}

func (wst *websocketTransport) Close() error {
	return wst.conn.Close()
}

func (wst *websocketTransport) StartTLS(cfg *tls.Config) {
}

func (wst *websocketTransport) EnableCompression(level compress.Level) {
}

func (wst *websocketTransport) ChannelBindingBytes(mechanism ChannelBindingMechanism) []byte {
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

func (wst *websocketTransport) readFromConn() error {
	if wst.r != nil && wst.r.Len() > 0 {
		return nil // remaining bytes in buffer...
	}
	_, r, err := wst.conn.NextReader()
	if err != nil {
		return err
	}
	n, err := r.Read(wst.rbuf)
	if err != nil {
		return err
	}
	if n > wst.maxStanzaSize {
		return ErrTooLargeStanza
	}
	wst.r = bytes.NewReader(wst.rbuf[:n])
	wst.p = xml.NewParser(wst.r)
	return nil
}
