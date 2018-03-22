/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ortuman/jackal/config"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/require"
)

func TestDocParse(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a xmlns="im.jackal">Hi!</a>\n`
	p := xml.NewParserTransportType(strings.NewReader(docSrc), config.SocketTransportType)
	a, err := p.ParseElement()
	require.Nil(t, err)
	require.NotNil(t, a)
	require.Equal(t, "a", a.Name())
	require.Equal(t, "im.jackal", a.Attribute("xmlns"))
	require.Equal(t, "Hi!", a.Text())
}

func TestEmptyDocParse(t *testing.T) {
	p := xml.NewParserTransportType(new(bytes.Buffer), config.SocketTransportType)
	_, err := p.ParseElement()
	require.NotNil(t, err)
}

func TestFailedDocParse(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a><b><c a="attr1">HI</c><b></a>\n`
	p := xml.NewParserTransportType(strings.NewReader(docSrc), config.SocketTransportType)
	_, err := p.ParseElement()
	require.NotNil(t, err)

	docSrc2 := `<?xml version="1.0" encoding="UTF-8"?>\n<element a="attr1">\n`
	p = xml.NewParserTransportType(strings.NewReader(docSrc2), config.SocketTransportType)
	element, err := p.ParseElement()
	require.Equal(t, io.EOF, err)
	require.Nil(t, element)
}

func TestParseSeveralElements(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?><a/>\n<b/>\n<c/>`
	reader := strings.NewReader(docSrc)
	p := xml.NewParserTransportType(reader, config.SocketTransportType)
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

func TestDocChildElements(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<parent><a/><b/><c/></parent>\n`
	p := xml.NewParserTransportType(strings.NewReader(docSrc), config.SocketTransportType)
	parent, err := p.ParseElement()
	require.Nil(t, err)
	require.NotNil(t, parent)
	childs := parent.Elements()
	require.Equal(t, 3, len(childs))
	require.Equal(t, "a", childs[0].Name())
	require.Equal(t, "b", childs[1].Name())
	require.Equal(t, "c", childs[2].Name())
}

func TestStream(t *testing.T) {
	openStreamXML := `<stream:stream xmlns:stream="http://etherx.jabber.org/streams" version="1.0" xmlns="jabber:client" to="localhost" xml:lang="en" xmlns:xml="http://www.w3.org/XML/1998/namespace"> `
	p := xml.NewParserTransportType(strings.NewReader(openStreamXML), config.SocketTransportType)
	elem, err := p.ParseElement()
	require.Nil(t, err)
	require.Equal(t, "stream:stream", elem.Name())
	closeStreamXML := `</stream:stream> `
	p = xml.NewParserTransportType(strings.NewReader(closeStreamXML), config.SocketTransportType)
	_, err = p.ParseElement()
	require.Equal(t, xml.ErrStreamClosedByPeer, err)
}
