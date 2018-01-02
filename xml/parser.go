/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

const rootElementIndex = -1

const streamName = "stream"

var ErrStreamClosedByPeer = errors.New("stream closed by peer")

// Parser parses arbitrary XML input and builds an array with the structure of all tag and data elements.
type Parser struct {
	nextElement  *Element
	parsingIndex int
	parsingStack []*MutableElement
	inElement    bool
}

// NewParser creates an empty Parser instance.
func NewParser() *Parser {
	p := &Parser{}
	p.parsingIndex = rootElementIndex
	p.parsingStack = make([]*MutableElement, 0)
	return p
}

// ParseElement parses next available XML element from reader.
func (p *Parser) ParseElement(reader io.Reader) (*Element, error) {
	d := xml.NewDecoder(reader)
	t, err := d.RawToken()
	if err != nil {
		return nil, err
	}
	for {
		switch t1 := t.(type) {
		case xml.StartElement:
			p.startElement(t1)
			if t1.Name.Local == streamName && t1.Name.Space == streamName {
				p.closeElement()
				goto done
			}

		case xml.CharData:
			p.setElementText(t1)

		case xml.EndElement:
			if t1.Name.Local == streamName && t1.Name.Space == streamName {
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
	return ret, nil
}

func (p *Parser) startElement(t xml.StartElement) {
	var name string
	if len(t.Name.Space) > 0 {
		name = fmt.Sprintf("%s:%s", t.Name.Space, t.Name.Local)
	} else {
		name = t.Name.Local
	}

	attrs := []Attribute{}
	for _, a := range t.Attr {
		name := xmlName(a.Name.Space, a.Name.Local)
		attrs = append(attrs, Attribute{name, a.Value})
	}
	element := NewMutableElementAttributes(name, attrs)
	p.parsingStack = append(p.parsingStack, element)
	p.parsingIndex++
	p.inElement = true
}

func (p *Parser) setElementText(t xml.CharData) {
	if !p.inElement {
		return
	}
	p.parsingStack[p.parsingIndex].SetText(string(t))
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
		p.nextElement = element.Copy()
	} else {
		p.parsingStack[p.parsingIndex].AppendElement(element.Copy())
	}
	p.inElement = false
}

func xmlName(space, local string) string {
	if len(space) > 0 {
		return fmt.Sprintf("%s:%s", space, local)
	}
	return local
}
