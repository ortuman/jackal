/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package stream

import (
	"sync"
	"testing"

	"sync/atomic"
	"time"

	"github.com/ortuman/jackal/xml"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestContext_Object(t *testing.T) {
	c := NewContext()
	require.Nil(t, c.Object("obj"))
	e := xml.NewElementName("presence")
	c.SetObject(e, "obj")
	require.Equal(t, e, c.Object("obj"))
}

func TestContext_String(t *testing.T) {
	c := NewContext()
	require.Equal(t, "", c.String("str"))
	s := "Hi world!"
	c.SetString(s, "str")
	require.Equal(t, s, c.String("str"))
}

func TestContext_Int(t *testing.T) {
	c := NewContext()
	require.Equal(t, 0, c.Int("int"))
	c.SetInt(178, "int")
	require.Equal(t, 178, c.Int("int"))
}

func TestContext_Float(t *testing.T) {
	c := NewContext()
	require.Equal(t, 0.0, c.Float("flt"))
	f := 3.141516
	c.SetFloat(f, "flt")
	require.Equal(t, f, c.Float("flt"))
}

func TestContext_Bool(t *testing.T) {
	c := NewContext()
	require.False(t, c.Bool("b"))
	c.SetBool(true, "b")
	require.True(t, c.Bool("b"))
}

func TestContext_DoOnce(t *testing.T) {
	var cnt int
	f := func() { cnt++ }
	h := uuid.New()
	c := NewContext()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() { c.DoOnce(h, f); wg.Done() }()
	}
	wg.Wait()
	require.Equal(t, 1, cnt)
}

func TestContext_Terminate(t *testing.T) {
	var cnt uint32

	var wg sync.WaitGroup
	c := NewContext()

	wg.Add(1)
	go func(doneCh <-chan struct{}) {
		select {
		case <-doneCh:
			atomic.AddUint32(&cnt, 1)
		case <-time.After(time.Second):
			return
		}
		wg.Done()
	}(c.Done())

	wg.Add(1)
	go func(doneCh <-chan struct{}) {
		select {
		case <-doneCh:
			atomic.AddUint32(&cnt, 1)
		case <-time.After(time.Second):
			break
		}
		wg.Done()
	}(c.Done())

	c.Terminate()
	wg.Wait()

	require.Equal(t, uint32(2), cnt)
}
