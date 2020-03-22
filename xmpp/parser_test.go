/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ortuman/jackal/xmpp"
	"github.com/stretchr/testify/require"
)

func TestDocParse(t *testing.T) {
	docSrc := `<a xmlns="im.jackal">Hi!</a>\n`
	p := xmpp.NewParser(strings.NewReader(docSrc), xmpp.DefaultMode, 0)
	a, err := p.ParseElement()
	require.Nil(t, err)
	require.NotNil(t, a)
	require.Equal(t, "a", a.Name())
	require.Equal(t, "im.jackal", a.Attributes().Get("xmlns"))
	require.Equal(t, "Hi!", a.Text())
}

func TestParser_EmptyDocParse(t *testing.T) {
	p := xmpp.NewParser(new(bytes.Buffer), xmpp.DefaultMode, 0)
	_, err := p.ParseElement()
	require.NotNil(t, err)
}

func TestParser_FailedDocParse(t *testing.T) {
	docSrc := `<a><b><c a="attr1">HI</c><b></a>\n`
	p := xmpp.NewParser(strings.NewReader(docSrc), xmpp.DefaultMode, 0)
	_, err := p.ParseElement()
	require.NotNil(t, err)

	docSrc2 := `<element a="attr1">\n`
	p = xmpp.NewParser(strings.NewReader(docSrc2), xmpp.DefaultMode, 0)
	element, err := p.ParseElement()
	require.Equal(t, io.EOF, err)
	require.Nil(t, element)

	docSrc3 := `</auth>\n`
	p = xmpp.NewParser(strings.NewReader(docSrc3), xmpp.DefaultMode, 0)
	element, err = p.ParseElement()
	require.NotNil(t, err)
	require.Nil(t, element)
}

func TestParser_Close(t *testing.T) {
	src := `</stream:stream>\n`
	p := xmpp.NewParser(strings.NewReader(src), xmpp.SocketStream, 0)
	_, err := p.ParseElement()
	require.Equal(t, xmpp.ErrStreamClosedByPeer, err)
}

func TestParser_ParseSeveralElements(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?><a/><b/><c/>`
	reader := strings.NewReader(docSrc)
	p := xmpp.NewParser(reader, xmpp.DefaultMode, 0)
	header, err := p.ParseElement()
	require.Nil(t, header)
	require.Nil(t, err)
	a, err := p.ParseElement()
	require.NotNil(t, a)
	require.Nil(t, err)
	b, err := p.ParseElement()
	require.NotNil(t, b)
	require.Nil(t, err)
	c, err := p.ParseElement()
	require.NotNil(t, c)
	require.Nil(t, err)
}

func TestParser_DocChildElements(t *testing.T) {
	docSrc := `<parent><a/><b/><c/></parent>\n`
	p := xmpp.NewParser(strings.NewReader(docSrc), xmpp.DefaultMode, 0)
	parent, err := p.ParseElement()
	require.Nil(t, err)
	require.NotNil(t, parent)
	childs := parent.Elements().All()
	require.Equal(t, 3, len(childs))
	require.Equal(t, "a", childs[0].Name())
	require.Equal(t, "b", childs[1].Name())
	require.Equal(t, "c", childs[2].Name())
}

func TestStream(t *testing.T) {
	openStreamXML := `<stream:stream xmlns:stream="http://etherx.jabber.org/streams" version="1.0" xmlns="jabber:client" to="localhost" xml:lang="en" xmlns:xml="http://www.w3.org/XML/1998/namespace"> `
	p := xmpp.NewParser(strings.NewReader(openStreamXML), xmpp.SocketStream, 0)
	elem, err := p.ParseElement()
	require.Nil(t, err)
	require.Equal(t, "stream:stream", elem.Name())
	closeStreamXML := `</stream:stream> `
	p = xmpp.NewParser(strings.NewReader(closeStreamXML), xmpp.SocketStream, 0)
	_, err = p.ParseElement()
	require.Equal(t, xmpp.ErrStreamClosedByPeer, err)
}
