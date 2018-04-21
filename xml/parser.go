/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/ortuman/jackal/config"
)

const rootElementIndex = -1

const (
	closeName  = "close"
	streamName = "stream"
)

const framedStreamNS = "urn:ietf:params:xml:ns:xmpp-framing"

// ErrStreamClosedByPeer is returned by Parse when peer closes the stream.
var ErrStreamClosedByPeer = errors.New("stream closed by peer")

// Parser parses arbitrary XML input and builds an array with the structure of all tag and data elements.
type Parser struct {
	tt           config.TransportType
	dec          *xml.Decoder
	nextElement  *Element
	parsingIndex int
	parsingStack []*Element
	inElement    bool
}

// NewParser creates an empty Parser instance.
func NewParser(reader io.Reader) *Parser {
	return &Parser{dec: xml.NewDecoder(reader), parsingIndex: rootElementIndex}
}

// NewParserTransportType creates an empty Parser instance associated to a transport type.
func NewParserTransportType(reader io.Reader, tt config.TransportType) *Parser {
	return &Parser{tt: tt, dec: xml.NewDecoder(reader), parsingIndex: rootElementIndex}
}

// ParseElement parses next available XML element from reader.
func (p *Parser) ParseElement() (XElement, error) {
	d := p.dec
	t, err := d.RawToken()
	if err != nil {
		return nil, err
	}
	for {
		switch t1 := t.(type) {
		case xml.ProcInst:
			return nil, nil

		case xml.StartElement:
			p.startElement(t1)
			if p.tt == config.SocketTransportType && t1.Name.Local == streamName && t1.Name.Space == streamName {
				p.closeElement()
				goto done
			}

		case xml.CharData:
			if !p.inElement {
				return nil, nil
			}
			p.setElementText(t1)

		case xml.EndElement:
			if p.tt == config.SocketTransportType && t1.Name.Local == streamName && t1.Name.Space == streamName {
				return nil, ErrStreamClosedByPeer
			}
			p.endElement(t1)
			if p.parsingIndex == rootElementIndex {
				goto done
			}
		}
		t, err = d.RawToken()
		if err != nil {
			return nil, err
		}
	}
done:
	ret := p.nextElement
	p.nextElement = nil
	if p.tt == config.WebSocketTransportType && ret.Name() == closeName && ret.Namespace() == framedStreamNS {
		return nil, ErrStreamClosedByPeer
	}
	return ret, nil
}

func (p *Parser) startElement(t xml.StartElement) {
	var name string
	if len(t.Name.Space) > 0 {
		name = fmt.Sprintf("%s:%s", t.Name.Space, t.Name.Local)
	} else {
		name = t.Name.Local
	}

	var attrs []Attribute
	for _, a := range t.Attr {
		name := xmlName(a.Name.Space, a.Name.Local)
		attrs = append(attrs, Attribute{name, a.Value})
	}
	element := &Element{name: name, attrs: attributeSet(attrs)}
	p.parsingStack = append(p.parsingStack, element)
	p.parsingIndex++
	p.inElement = true
}

func (p *Parser) setElementText(t xml.CharData) {
	elem := p.parsingStack[p.parsingIndex]
	elem.text = string(t)
}

func (p *Parser) endElement(t xml.EndElement) error {
	name := xmlName(t.Name.Space, t.Name.Local)
	if p.parsingStack[p.parsingIndex].Name() != name {
		return fmt.Errorf("unexpected end element </" + name + ">")
	}
	p.closeElement()
	return nil
}

func (p *Parser) closeElement() {
	element := p.parsingStack[p.parsingIndex]
	p.parsingStack = p.parsingStack[:p.parsingIndex]

	p.parsingIndex--
	if p.parsingIndex == rootElementIndex {
		p.nextElement = element
	} else {
		p.parsingStack[p.parsingIndex].AppendElement(element)
	}
	p.inElement = false
}

func xmlName(space, local string) string {
	if len(space) > 0 {
		return fmt.Sprintf("%s:%s", space, local)
	}
	return local
}
