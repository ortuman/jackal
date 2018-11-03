/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package app

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/ortuman/jackal/version"
	"github.com/stretchr/testify/require"
)

type writerBuffer struct {
	mu      sync.RWMutex
	buf     *bytes.Buffer
	closeCh chan bool
}

func newWriterBuffer() *writerBuffer {
	return &writerBuffer{buf: bytes.NewBuffer(nil), closeCh: make(chan bool)}
}

func (wb *writerBuffer) Write(p []byte) (int, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	return wb.buf.Write(p)
}

func (wb *writerBuffer) String() string {
	wb.mu.RLock()
	defer wb.mu.RUnlock()
	return wb.buf.String()
}

func TestApplicationEmptyArgs(t *testing.T) {
	require.NotNil(t, New(nil, nil))
}

func TestApplicationShowUsage(t *testing.T) {
	w := newWriterBuffer()
	c, err := New(w, []string{"./jackal", "-h"}).Run()
	require.Nil(t, err)
	require.Equal(t, successCode, c)
	require.Equal(t, expectedUsageString(), w.String())
}

func TestApplicationPrintVersion(t *testing.T) {
	w := newWriterBuffer()
	args := []string{"./jackal", "--version"}
	c, err := New(w, args).Run()
	require.Nil(t, err)
	require.Equal(t, successCode, c)
	require.Equal(t, fmt.Sprintf("jackal version: %v\n", version.ApplicationVersion), w.String())
}

func TestApplication_Run(t *testing.T) {
	w := newWriterBuffer()
	args := []string{"./jackal", "--config=../testdata/config_basic.yml"}
	ap := New(w, args)
	go func() {
		time.Sleep(time.Millisecond * 1500) // wait until initialized
		ap.waitStopCh <- syscall.SIGTERM
	}()
	ap.shutDownWaitSecs = time.Duration(2) * time.Second // wait only two seconds
	c, err := ap.Run()
	require.Nil(t, err)
	require.Equal(t, successCode, c)

	os.RemoveAll(".cert/")

	// make sure pid and log files had been created
	_, err = os.Stat("test.jackal.pid")
	require.False(t, os.IsNotExist(err))
	os.Remove("test.jackal.pid")

	_, err = os.Stat("test.jackal.log")
	require.False(t, os.IsNotExist(err))
	os.Remove("test.jackal.log")
}

func expectedUsageString() string {
	var r string
	for i := range logoStr {
		r += fmt.Sprintf("%s\n", logoStr[i])
	}
	r += fmt.Sprintf("%s\n", usageStr)
	return r
}
