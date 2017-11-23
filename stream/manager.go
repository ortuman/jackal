/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package stream

import "sync"

type StreamManager struct {
	strms       map[string]*Stream
	authedStrms map[string]*Stream

	regCh   chan *Stream
	unregCh chan *Stream
	authCh  chan *Stream
}

// singleton interface
var (
	instance *StreamManager
	once     sync.Once
)

func Manager() *StreamManager {
	once.Do(func() {
		instance = &StreamManager{
			strms:       make(map[string]*Stream),
			authedStrms: make(map[string]*Stream),
			regCh:       make(chan *Stream),
			unregCh:     make(chan *Stream),
			authCh:      make(chan *Stream),
		}
		go instance.loop()
	})
	return instance
}

func (m *StreamManager) RegisterStream(strm *Stream) {
	m.regCh <- strm
}

func (m *StreamManager) UnregisterStream(strm *Stream) {
	m.unregCh <- strm
}

func (m *StreamManager) AuthenticateStream(strm *Stream) {
	m.authCh <- strm
}

func (m *StreamManager) loop() {
}
