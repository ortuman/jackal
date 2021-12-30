// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xmppparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_ErrTooLargeStanzaRead(t *testing.T) {
	// given
	docSrc := `<a/><be/>`
	p := New(strings.NewReader(docSrc), SocketStream, 4)

	// when
	a, err0 := p.Parse()
	be, err1 := p.Parse()

	// then
	require.Nil(t, err0)
	require.NotNil(t, a)
	require.Equal(t, "<a/>", a.String())

	require.Nil(t, be)
	require.Equal(t, ErrTooLargeStanza, err1)
}

func TestParser_ParseSeveralElements(t *testing.T) {
	// given
	docSrc := `<?xml version="1.0" encoding="UTF-8"?><a/><b/><c/>`

	r := strings.NewReader(docSrc)
	p := New(r, DefaultMode, 1024)

	// when
	a, err1 := p.Parse()
	b, err2 := p.Parse()
	c, err3 := p.Parse()

	// then
	require.NotNil(t, a)
	require.Nil(t, err1)

	require.NotNil(t, b)
	require.Nil(t, err2)

	require.NotNil(t, c)
	require.Nil(t, err3)
}

func TestParser_DocChildElements(t *testing.T) {
	// given
	docSrc := `<parent><a/><b/><c/></parent>\n`
	p := New(strings.NewReader(docSrc), DefaultMode, 1024)

	// when
	elem, err := p.Parse()
	require.Nil(t, err)
	require.NotNil(t, elem)

	childs := elem.AllChildren()

	// then
	require.Equal(t, 3, len(childs))
	require.Equal(t, "a", childs[0].Name())
	require.Equal(t, "b", childs[1].Name())
	require.Equal(t, "c", childs[2].Name())
}

func TestParser_Stream(t *testing.T) {
	openStreamXML := `<stream:stream xmlns:stream="http://etherx.jabber.org/streams" version="1.0" xmlns="jabber:client" to="localhost" xml:lang="en" xmlns:xml="http://www.w3.org/XML/1998/namespace"> `
	p := New(strings.NewReader(openStreamXML), SocketStream, 1024)
	elem, err := p.Parse()
	require.Nil(t, err)
	require.Equal(t, "stream:stream", elem.Name())

	closeStreamXML := `</stream:stream> `
	p = New(strings.NewReader(closeStreamXML), SocketStream, 1024)

	_, err = p.Parse()

	require.Equal(t, ErrStreamClosedByPeer, err)
}
