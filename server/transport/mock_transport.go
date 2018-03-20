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

	"github.com/ortuman/jackal/config"
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
}

// NewMockTransport returns a new MockTransport instance.
func NewMockTransport() *MockTransport {
	tr := &MockTransport{}
	tr.wb = new(bytes.Buffer)
	tr.rb = new(bytes.Buffer)
	tr.br = bufio.NewReader(tr.rb)
	tr.bw = bufio.NewWriter(tr.wb)
	return tr
}

// Read reads a byte array from the mocked transport.
func (mt *MockTransport) Read(p []byte) (n int, err error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	return mt.br.Read(p)
}

// SetReadBytes sets transport next read operation result.
func (mt *MockTransport) SetReadBytes(p []byte) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.rb.Reset()
	mt.rb.Write(p)
}

// Write writes a byte array to the mocked transport internal buffer.
func (mt *MockTransport) Write(p []byte) (n int, err error) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	defer mt.bw.Flush()
	return mt.bw.Write(p)
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
func (mt *MockTransport) EnableCompression(level config.CompressionLevel) {
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
func (mt *MockTransport) ChannelBindingBytes(config.ChannelBindingMechanism) []byte {
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
