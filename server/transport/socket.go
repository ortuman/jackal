/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bufio"
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
	w                  io.Writer
	r                  io.Reader
	br                 *bufio.Reader
	bw                 *bufio.Writer
	readTimeout        int
	compressionEnabled bool
	parser             *xml.Parser
}

// NewSocketTransport creates a socket class stream transport.
func NewSocketTransport(conn net.Conn, bufferSize, keepAlive int) Transport {
	s := &socketTransport{
		conn:        conn,
		br:          bufio.NewReaderSize(conn, bufferSize),
		bw:          bufio.NewWriterSize(conn, bufferSize),
		readTimeout: keepAlive,
	}
	s.w = s.bw
	s.r = s.br
	s.parser = xml.NewParser(s.r)
	return s
}

func (s *socketTransport) ReadElement() (xml.Element, error) {
	s.conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(s.readTimeout)))
	return s.parser.ParseElement()
}

func (s *socketTransport) WriteString(str string) error {
	defer s.bw.Flush()
	_, err := io.Copy(s.w, strings.NewReader(str))
	return err
}

func (s *socketTransport) WriteElement(elem xml.Element, includeClosing bool) error {
	defer s.bw.Flush()
	elem.ToXML(s.w, includeClosing)
	return nil
}

func (s *socketTransport) Close() error {
	return s.conn.Close()
}

func (s *socketTransport) StartTLS(cfg *tls.Config) {
	if _, ok := s.conn.(*tls.Conn); !ok {
		s.conn = tls.Server(s.conn, cfg)
		s.bw.Reset(s.conn)
		s.br.Reset(s.conn)
		s.parser = xml.NewParser(s.r)
	}
}

func (s *socketTransport) EnableCompression(level config.CompressionLevel) {
	if !s.compressionEnabled {
		zwr := compress.NewZlibCompressor(s.br, s.bw, level)
		s.w = zwr
		s.r = zwr
		s.parser = xml.NewParser(s.r)
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
