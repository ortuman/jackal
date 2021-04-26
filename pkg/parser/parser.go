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
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/jackal-xmpp/stravaganza/v2"
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

// ErrNoElement will be returned by Parse when no elements are available to be parsed in the reader buffer stream.
var ErrNoElement = errors.New("parser: no elements")

// Parser parses arbitrary XML input and builds an array with the structure of all tag and data elements.
type Parser struct {
	mode              ParsingMode
	r                 io.Reader
	rdBuf             []byte
	dec               *xml.Decoder
	nextElement       stravaganza.Element
	stack             []*stravaganza.Builder
	index             int
	inElement         bool
	rootElementOffset int64
	maxStanzaSize     int64
}

// New creates an empty Parser instance.
func New(reader io.Reader, mode ParsingMode, maxStanzaSize int) *Parser {
	return &Parser{
		mode:          mode,
		r:             reader,
		rdBuf:         make([]byte, maxStanzaSize+1),
		index:         rootElementIndex,
		maxStanzaSize: int64(maxStanzaSize),
	}
}

// Parse parses next available XML element from reader.
func (p *Parser) Parse() (stravaganza.Element, error) {
	if p.dec == nil {
		n, err := p.r.Read(p.rdBuf)
		switch {
		case err != nil:
			return nil, err
		case n == 0:
			return nil, ErrNoElement
		case int64(n) > p.maxStanzaSize:
			return nil, ErrTooLargeStanza
		}
		p.dec = xml.NewDecoder(bytes.NewReader(p.rdBuf[:n]))
	}
	// parse input buffer stream
	for {
		t, err := p.dec.RawToken()
		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
				p.dec = nil
				return nil, ErrNoElement

			default:
				return nil, err
			}
		}
		if p.index != rootElementIndex && p.dec.InputOffset()-p.rootElementOffset > p.maxStanzaSize {
			return nil, ErrTooLargeStanza
		}
		switch t1 := t.(type) {
		case xml.ProcInst:
			break

		case xml.CharData:
			if p.inElement {
				p.setElementText(t1)
			}
			break

		case xml.StartElement:
			p.startElement(t1)
			if p.mode == SocketStream && t1.Name.Local == streamName && t1.Name.Space == streamName {
				if err := p.closeElement(xmlName(t1.Name.Space, t1.Name.Local)); err != nil {
					return nil, err
				}
				return p.popElement(), nil
			}
			break

		case xml.EndElement:
			if p.mode == SocketStream && t1.Name.Local == streamName && t1.Name.Space == streamName {
				return nil, ErrStreamClosedByPeer
			}
			if err := p.endElement(t1); err != nil {
				return nil, err
			}
			if p.index == rootElementIndex {
				return p.popElement(), nil
			}
			break
		}
	}
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

	if p.index == rootElementIndex {
		p.rootElementOffset = p.dec.InputOffset()
	}
	p.index = len(p.stack) - 1
	p.inElement = true
}

func (p *Parser) setElementText(t xml.CharData) {
	p.stack[p.index] = p.stack[p.index].WithText(string(t))
}

func (p *Parser) endElement(t xml.EndElement) error {
	return p.closeElement(xmlName(t.Name.Space, t.Name.Local))
}

func (p *Parser) closeElement(name string) error {
	if p.index == rootElementIndex {
		return errUnexpectedEnd(name)
	}
	builder := p.stack[p.index]
	p.stack = p.stack[:p.index]

	element := builder.Build()

	if name != element.Name() {
		return errUnexpectedEnd(name)
	}
	p.index = len(p.stack) - 1
	if p.index == rootElementIndex {
		p.nextElement = element
		p.rootElementOffset = 0
	} else {
		p.stack[p.index] = p.stack[p.index].WithChild(element)
	}
	p.inElement = false
	return nil
}

func (p *Parser) popElement() stravaganza.Element {
	elem := p.nextElement
	p.nextElement = nil
	return elem
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
