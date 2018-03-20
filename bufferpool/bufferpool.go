/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package bufferpool

import (
	"bytes"
	"sync"
)

// Pool represents a buffer pool container.
type Pool struct {
	pool sync.Pool
}

// New returns a new buffer pool instance.
func New() *Pool {
	p := Pool{
		pool: sync.Pool{New: func() interface{} {
			return new(bytes.Buffer)
		},
		},
	}
	return &p
}

// Get returns a buffer instance from the pool.
func (p *Pool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

// Put returns a buffer instance to the pool.
func (p *Pool) Put(buf *bytes.Buffer) {
	buf.Reset()
	p.pool.Put(buf)
}
