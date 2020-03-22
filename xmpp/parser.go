/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

const rootElementIndex = -1

const (
	streamName = "stream"
)

// ParsingMode defines the way in which special parsed element
// should be considered or not according to the reader nature.
type ParsingMode int

const (
	// DefaultMode treats incoming elements as provided from raw byte reader.
	DefaultMode = ParsingMode(iota)

	// SocketStream treats incoming elements as provided from a socket transport.
	SocketStream
)

// ErrTooLargeStanza is returned by ReadElement when the size of
// the incoming stanza is too large.
var ErrTooLargeStanza = errors.New("xml: too large stanza")

// ErrStreamClosedByPeer is returned by Parse when peer closes the stream.
var ErrStreamClosedByPeer = errors.New("xml: stream closed by peer")

// Parser parses arbitrary XML input and builds an array with the structure of all tag and data elements.
type Parser struct {
	dec           *xml.Decoder
	mode          ParsingMode
	nextElement   *Element
	parsingIndex  int
	parsingStack  []*Element
	inElement     bool
	lastOffset    int64
	maxStanzaSize int64
}

// NewParser creates an empty Parser instance.
func NewParser(reader io.Reader, mode ParsingMode, maxStanzaSize int) *Parser {
	return &Parser{
		dec:           xml.NewDecoder(reader),
		mode:          mode,
		parsingIndex:  rootElementIndex,
		maxStanzaSize: int64(maxStanzaSize),
	}
}

// ParseElement parses next available XML element from reader.
func (p *Parser) ParseElement() (XElement, error) {
	t, err := p.dec.RawToken()
	if err != nil {
		return nil, err
	}
	for {
		// check max stanza size limit
		off := p.dec.InputOffset()
		if p.maxStanzaSize > 0 && off-p.lastOffset > p.maxStanzaSize {
			return nil, ErrTooLargeStanza
		}
		switch t1 := t.(type) {
		case xml.ProcInst:
			return nil, nil

		case xml.StartElement:
			p.startElement(t1)
			if p.mode == SocketStream && t1.Name.Local == streamName && t1.Name.Space == streamName {
				p.closeElement()
				goto done
			}

		case xml.CharData:
			if !p.inElement {
				return nil, nil
			}
			p.setElementText(t1)

		case xml.EndElement:
			if p.mode == SocketStream && t1.Name.Local == streamName && t1.Name.Space == streamName {
				return nil, ErrStreamClosedByPeer
			}
			if err := p.endElement(t1); err != nil {
				return nil, err
			}
			if p.parsingIndex == rootElementIndex {
				goto done
			}
		}
		t, err = p.dec.RawToken()
		if err != nil {
			return nil, err
		}
	}
done:
	p.lastOffset = p.dec.InputOffset()
	ret := p.nextElement

	p.nextElement = nil
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
	p.parsingIndex = len(p.parsingStack) - 1
	p.inElement = true
}

func (p *Parser) setElementText(t xml.CharData) {
	elem := p.parsingStack[p.parsingIndex]
	elem.text = string(t)
}

func (p *Parser) endElement(t xml.EndElement) error {
	name := xmlName(t.Name.Space, t.Name.Local)
	if p.parsingIndex == rootElementIndex || p.parsingStack[p.parsingIndex].Name() != name {
		return fmt.Errorf("unexpected end element </" + name + ">")
	}
	p.closeElement()
	return nil
}

func (p *Parser) closeElement() {
	element := p.parsingStack[p.parsingIndex]
	p.parsingStack = p.parsingStack[:p.parsingIndex]

	p.parsingIndex = len(p.parsingStack) - 1
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
