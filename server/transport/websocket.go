/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"crypto/tls"
	"io"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/xml"
)

type websocketTransport struct {
	conn        *websocket.Conn
	readTimeout int
}

// NewSocketTransport creates a socket class stream transport.
func NewWebSocketTransport(conn *websocket.Conn, keepAlive int) Transport {
	wst := &websocketTransport{conn: conn, readTimeout: keepAlive}
	return wst
}

func (wst *websocketTransport) ReadElement() (xml.Element, error) {
	wst.conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(wst.readTimeout)))
	_, r, err := wst.conn.NextReader()
	if err != nil {
		return nil, err
	}
	p := xml.NewParserTransportType(r, config.WebSocketTransportType)
	return p.ParseElement()
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

func (wst *websocketTransport) WriteElement(elem xml.Element, includeClosing bool) error {
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

func (wst *websocketTransport) EnableCompression(level config.CompressionLevel) {
}

func (wst *websocketTransport) ChannelBindingBytes(mechanism config.ChannelBindingMechanism) []byte {
	if tlsConn, ok := wst.conn.UnderlyingConn().(*tls.Conn); ok {
		switch mechanism {
		case config.TLSUnique:
			st := tlsConn.ConnectionState()
			return st.TLSUnique
		default:
			break
		}
	}
	return nil
}
