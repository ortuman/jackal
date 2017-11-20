/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import (
	"encoding/xml"
	"io"
)

const rootElementIndex = -1

type Parser struct {
	elements     []*Element
	parsingIndex int
	parsingStack []*MutableElement
	inElement    bool
}

func NewParser() *Parser {
	p := &Parser{}
	p.elements = make([]*Element, 0)
	p.parsingIndex = rootElementIndex
	p.parsingStack = make([]*MutableElement, 0)
	return p
}

func (p *Parser) ParseElements(reader io.Reader) error {
	d := xml.NewDecoder(reader)
	t, err := d.Token()
	for t != nil {
		switch t1 := t.(type) {
		case xml.StartElement:
			p.startElement(t1)
		case xml.CharData:
			p.setElementText(t1)
		case xml.EndElement:
			p.endElement(t1)
		}
		t, err = d.Token()
	}
	if err != nil && err != io.EOF {
		return err
	}
	return nil
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
	name := t.Name.Local
	attrs := []Attribute{}
	for _, a := range t.Attr {
		attrs = append(attrs, Attribute{a.Name.Local, a.Value})
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

func (p *Parser) endElement(t xml.EndElement) {
	element := p.parsingStack[len(p.parsingStack)-1]
	p.parsingStack = p.parsingStack[:len(p.parsingStack)-1]

	p.parsingIndex--
	if p.parsingIndex == rootElementIndex {
		p.elements = append(p.elements, element.Copy())
	} else {
		p.parsingStack[p.parsingIndex].AppendElement(element.Copy())
	}
	p.inElement = false
}
