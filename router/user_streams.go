package router

import (
	"sync"

	"github.com/ortuman/jackal/stream"
)

type userStreamList struct {
	mu      sync.RWMutex
	streams []stream.C2S
}

func (usl *userStreamList) bind(stm stream.C2S) int {
	usl.mu.Lock()
	defer usl.mu.Unlock()

	res := stm.Resource()
	for _, stm := range usl.streams {
		if stm.Resource() == res {
			// already binded
			return len(usl.streams)
		}
	}
	usl.streams = append(usl.streams, stm)
	return len(usl.streams)
}

func (usl *userStreamList) unbind(res string) int {
	usl.mu.Lock()
	defer usl.mu.Unlock()

	for i := 0; i < len(usl.streams); i++ {
		if res == usl.streams[i].Resource() {
			usl.streams = append(usl.streams[:i], usl.streams[i+1:]...)
			break
		}
	}
	return len(usl.streams)
}

func (usl *userStreamList) all() []stream.C2S {
	usl.mu.RLock()
	defer usl.mu.RUnlock()
	return usl.streams
}
