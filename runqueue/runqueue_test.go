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
		rq.Post(fn)
		if i%2 == 1 {
			time.Sleep(time.Millisecond)
		}
	}
	wg.Wait()

	require.Equal(t, int32(2000), i)
}
