/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
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

// Parser parses arbitrary XML input and builds an array with the structure of all tag and data elements.
type Parser struct {
	elements     []*Element
	parsingIndex int
	parsingStack []*MutableElement
	inElement    bool
}

// NewParser creates an empty Parser instance.
func NewParser() *Parser {
	p := &Parser{}
	p.elements = make([]*Element, 0)
	p.parsingIndex = rootElementIndex
	p.parsingStack = make([]*MutableElement, 0)
	return p
}

func (p *Parser) ParseElements(reader io.Reader) error {
	d := xml.NewDecoder(reader)
	t, err := d.RawToken()
	for t != nil {
		switch t1 := t.(type) {
		case xml.StartElement:
			p.startElement(t1)
		case xml.CharData:
			p.setElementText(t1)
		case xml.EndElement:
			if err := p.endElement(t1); err != nil {
				return err
			}
		}
		t, err = d.RawToken()
	}
	if err != nil && err != io.EOF {
		return err
	}
	switch p.parsingIndex {
	case 0:
		p.closeElement() // open stream element
		fallthrough
	case rootElementIndex:
		return nil
	default:
		return errors.New("malformed XML")
	}
}

func (p *Parser) PopElement() *Element {
	if len(p.elements) == 0 {
		return nil
	}
	element := p.elements[0]
	p.elements = append(p.elements[:0], p.elements[1:]...)
	return element
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
		name := xmlName(t.Name.Space, t.Name.Local)
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
		p.elements = append(p.elements, element.Copy())
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
