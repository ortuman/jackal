/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package pool

import (
	"bytes"
	"sync"
)

// BufferPool represents a buffer pool container.
type BufferPool struct {
	p sync.Pool
}

// NewBufferPool returns a new buffer pool instance.
func NewBufferPool() *BufferPool {
	bp := BufferPool{
		p: sync.Pool{New: func() interface{} { return new(bytes.Buffer) }},
	}
	return &bp
}

// Get returns a buffer instance from the pool.
func (bp *BufferPool) Get() *bytes.Buffer {
	return bp.p.Get().(*bytes.Buffer)
}

// Put returns a buffer instance to the pool.
func (bp *BufferPool) Put(buf *bytes.Buffer) {
	buf.Reset()
	bp.p.Put(buf)
}
