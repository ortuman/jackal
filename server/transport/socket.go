/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"strings"
	"time"

	"github.com/ortuman/jackal/config"
	"github.com/ortuman/jackal/server/compress"
	"github.com/ortuman/jackal/xml"
)

type socketTransport struct {
	conn               net.Conn
	rw                 io.ReadWriter
	bw                 *bufio.Writer
	r                  *bytes.Reader
	rbuf               []byte
	p                  *xml.Parser
	maxStanzaSize      int
	keepAlive          int
	compressionEnabled bool
}

// NewSocketTransport creates a socket class stream transport.
func NewSocketTransport(conn net.Conn, maxStanzaSize, keepAlive int) Transport {
	s := &socketTransport{
		conn:          conn,
		rw:            conn,
		bw:            bufio.NewWriter(conn),
		rbuf:          make([]byte, maxStanzaSize+1),
		maxStanzaSize: maxStanzaSize,
		keepAlive:     keepAlive,
	}
	return s
}

func (s *socketTransport) ReadElement() (xml.XElement, error) {
	if err := s.readFromConn(); err != nil {
		return nil, err
	}
	return s.p.ParseElement()
}

func (s *socketTransport) WriteString(str string) error {
	_, err := io.Copy(s.rw, strings.NewReader(str))
	return err
}

func (s *socketTransport) WriteElement(elem xml.XElement, includeClosing bool) error {
	defer s.bw.Flush()
	elem.ToXML(s.bw, includeClosing)
	return nil
}

func (s *socketTransport) Close() error {
	return s.conn.Close()
}

func (s *socketTransport) StartTLS(cfg *tls.Config) {
	if _, ok := s.conn.(*tls.Conn); !ok {
		s.conn = tls.Server(s.conn, cfg)
		s.rw = s.conn
		s.bw.Reset(s.rw)
		s.r = nil
	}
}

func (s *socketTransport) EnableCompression(level config.CompressionLevel) {
	if !s.compressionEnabled {
		s.rw = compress.NewZlibCompressor(s.rw, s.rw, level)
		s.bw.Reset(s.rw)
		s.r = nil
		s.compressionEnabled = true
	}
}

func (s *socketTransport) ChannelBindingBytes(mechanism config.ChannelBindingMechanism) []byte {
	if tlsConn, ok := s.conn.(*tls.Conn); ok {
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

func (s *socketTransport) readFromConn() error {
	if s.r != nil && s.r.Len() > 0 {
		return nil // remaining bytes in buffer...
	}
	s.conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(s.keepAlive)))
	n, err := s.rw.Read(s.rbuf)
	if err != nil {
		return err
	}
	if n > s.maxStanzaSize {
		return ErrTooLargeStanza
	}
	s.r = bytes.NewReader(s.rbuf[:n])
	s.p = xml.NewParserTransportType(s.r, config.SocketTransportType)
	return nil
}
