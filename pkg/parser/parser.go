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
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/jackal-xmpp/stravaganza"
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

// ErrTooLargeStanza will be returned Parse when the size of the incoming stanza is too large.
var ErrTooLargeStanza = errors.New("parser: too large stanza")

// ErrStreamClosedByPeer will be returned by Parse when stream closed element is parsed.
var ErrStreamClosedByPeer = errors.New("parser: stream closed by peer")

// Parser parses arbitrary XML input and builds an array with the structure of all tag and data elements.
type Parser struct {
	dec           *xml.Decoder
	mode          ParsingMode
	nextElement   stravaganza.Element
	stack         []*stravaganza.Builder
	pIndex        int
	inElement     bool
	lastOffset    int64
	maxStanzaSize int64
}

// New creates an empty Parser instance.
func New(reader io.Reader, mode ParsingMode, maxStanzaSize int) *Parser {
	return &Parser{
		mode:          mode,
		dec:           xml.NewDecoder(reader),
		pIndex:        rootElementIndex,
		maxStanzaSize: int64(maxStanzaSize),
	}
}

// Parse parses next available XML element from reader.
func (p *Parser) Parse() (stravaganza.Element, error) {
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
		case xml.StartElement:
			p.startElement(t1)
			if p.mode == SocketStream && t1.Name.Local == streamName && t1.Name.Space == streamName {
				if err := p.closeElement(xmlName(t1.Name.Space, t1.Name.Local)); err != nil {
					return nil, err
				}
				goto done
			}

		case xml.CharData:
			if p.inElement {
				p.setElementText(t1)
			}

		case xml.EndElement:
			if p.mode == SocketStream && t1.Name.Local == streamName && t1.Name.Space == streamName {
				return nil, ErrStreamClosedByPeer
			}
			if err := p.endElement(t1); err != nil {
				return nil, err
			}
			if p.pIndex == rootElementIndex {
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
	elem := p.nextElement
	p.nextElement = nil

	return elem, nil
}

func (p *Parser) startElement(t xml.StartElement) {
	name := xmlName(t.Name.Space, t.Name.Local)

	var attrs []stravaganza.Attribute
	for _, a := range t.Attr {
		name := xmlName(a.Name.Space, a.Name.Local)
		attrs = append(attrs, stravaganza.Attribute{Label: name, Value: a.Value})
	}
	builder := stravaganza.NewBuilder(name).WithAttributes(attrs...)
	p.stack = append(p.stack, builder)

	p.pIndex = len(p.stack) - 1
	p.inElement = true
}

func (p *Parser) setElementText(t xml.CharData) {
	p.stack[p.pIndex] = p.stack[p.pIndex].WithText(string(t))
}

func (p *Parser) endElement(t xml.EndElement) error {
	return p.closeElement(xmlName(t.Name.Space, t.Name.Local))
}

func (p *Parser) closeElement(name string) error {
	if p.pIndex == rootElementIndex {
		return errUnexpectedEnd(name)
	}
	builder := p.stack[p.pIndex]
	p.stack = p.stack[:p.pIndex]

	element := builder.Build()

	if name != element.Name() {
		return errUnexpectedEnd(name)
	}
	p.pIndex = len(p.stack) - 1
	if p.pIndex == rootElementIndex {
		p.nextElement = element
	} else {
		p.stack[p.pIndex] = p.stack[p.pIndex].WithChild(element)
	}
	p.inElement = false
	return nil
}

func xmlName(space, local string) string {
	if len(space) > 0 {
		return fmt.Sprintf("%s:%s", space, local)
	}
	return local
}

func errUnexpectedEnd(name string) error {
	return fmt.Errorf("xmppparser: unexpected end element </%s>", name)
}
