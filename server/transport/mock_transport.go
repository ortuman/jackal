/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package transport

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"sync"

	"github.com/ortuman/jackal/server/compress"
	"github.com/ortuman/jackal/xml"
)

// MockTransport represents a mocked transport type.
type MockTransport struct {
	mu            sync.RWMutex
	wb            *bytes.Buffer
	rb            *bytes.Buffer
	br            *bufio.Reader
	bw            *bufio.Writer
	cBindingBytes []byte
	closed        bool
	secured       bool
	compressed    bool
	parser        *xml.Parser
}

// NewMockTransport returns a new MockTransport instance.
func NewMockTransport() *MockTransport {
	mt := &MockTransport{}
	mt.wb = new(bytes.Buffer)
	mt.rb = new(bytes.Buffer)
	mt.br = bufio.NewReader(mt.rb)
	mt.bw = bufio.NewWriter(mt.wb)
	mt.parser = xml.NewParser(mt.br)
	return mt
}

// ReadElement reads next available XML element from the mocked transport.
func (mt *MockTransport) ReadElement() (xml.XElement, error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	return mt.parser.ParseElement()
}

// SetReadBytes sets transport next read operation result.
func (mt *MockTransport) SetReadBytes(p []byte) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.rb.Reset()
	mt.rb.Write(p)
}

// WriteString writes a raw string to the mocked transport.
func (mt *MockTransport) WriteString(str string) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	defer mt.bw.Flush()
	_, err := mt.bw.WriteString(str)
	return err
}

// WriteElement writes an XML element the mocked transport.
func (mt *MockTransport) WriteElement(elem xml.XElement, includeClosing bool) error {
	defer mt.bw.Flush()
	elem.ToXML(mt.bw, includeClosing)
	return nil
}

// GetWrittenBytes returns transport previously written bytes.
func (mt *MockTransport) GetWrittenBytes() []byte {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	defer mt.wb.Reset()
	return mt.wb.Bytes()
}

// Close marks a mocked transport as closed.
func (mt *MockTransport) Close() error {
	mt.mu.Lock()
	mt.closed = true
	mt.mu.Unlock()
	return nil
}

// IsClosed returns whether or not the mocked transport
// has been previously closed.
func (mt *MockTransport) IsClosed() bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.closed
}

// StartTLS secures the mocked transport.
func (mt *MockTransport) StartTLS(tlsCfg *tls.Config) {
	mt.mu.Lock()
	mt.secured = true
	mt.mu.Unlock()
}

// IsSecured returns whether or not the mocked transport
// has been previously secured.
func (mt *MockTransport) IsSecured() bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.secured
}

// EnableCompression marks a mocked transport as compressed.
func (mt *MockTransport) EnableCompression(level compress.Level) {
	mt.mu.Lock()
	mt.compressed = true
	mt.mu.Unlock()
}

// IsCompressed returns whether or not the mocked transport
// has been previously compressed.
func (mt *MockTransport) IsCompressed() bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.compressed
}

// ChannelBindingBytes returns mocked transport channel binding bytes.
func (mt *MockTransport) ChannelBindingBytes(ChannelBindingMechanism) []byte {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.cBindingBytes
}

// SetChannelBindingBytes sets mocked transport channel binding bytes.
func (mt *MockTransport) SetChannelBindingBytes(cBindingBytes []byte) {
	mt.mu.Lock()
	mt.cBindingBytes = cBindingBytes
	mt.mu.Unlock()
}
