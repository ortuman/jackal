package runqueue

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunQueueConsistency(t *testing.T) {
	var i int32

	var wg sync.WaitGroup
	fn := func() {
		i++
		wg.Done()
	}

	rq := New("test")

	for i := 0; i < 2000; i++ {
		wg.Add(1)
		rq.Run(fn)
		if i%2 == 1 {
			time.Sleep(time.Millisecond)
		}
	}
	wg.Wait()

	require.Equal(t, int32(2000), i)
}

func TestRunQueueStop(t *testing.T) {
	fn := func() {
		time.Sleep(time.Millisecond * 500)
	}
	rq := New("test")
	rq.Run(fn)

	c := make(chan struct{})
	rq.Stop(func() { close(c) })

	select {
	case <-c:
	case <-time.NewTimer(time.Second).C:
		require.Fail(t, "close channel timeout")
		break
	}
}
