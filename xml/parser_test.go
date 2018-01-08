/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml_test

import (
	"io"
	"strings"
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/assert"
)

func TestDocParse(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a xmlns="im.jackal">Hi!</a>\n`
	p := xml.NewParser(strings.NewReader(docSrc))
	a, err := p.ParseElement()
	assert.Nil(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, a.Name(), "a")
	assert.Equal(t, a.Attribute("xmlns"), "im.jackal")
	assert.Equal(t, a.Text(), "Hi!")
}

func TestFailedDocParse(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a><b><c a="attr1">HI</c><b></a>\n`
	p := xml.NewParser(strings.NewReader(docSrc))
	_, err := p.ParseElement()
	assert.NotNil(t, err)

	docSrc2 := `<?xml version="1.0" encoding="UTF-8"?>\n<element a="attr1">\n`
	p = xml.NewParser(strings.NewReader(docSrc2))
	element, err := p.ParseElement()
	assert.Equal(t, err, io.EOF)
	assert.Nil(t, element)
}

func TestParseSeveralElements(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?><a/>\n<b/>\n<c/>`
	reader := strings.NewReader(docSrc)
	p := xml.NewParser(reader)
	a, err := p.ParseElement()
	assert.NotNil(t, a)
	assert.Nil(t, err)
	b, err := p.ParseElement()
	assert.NotNil(t, b)
	assert.Nil(t, err)
	c, err := p.ParseElement()
	assert.NotNil(t, c)
	assert.Nil(t, err)
}

func TestDocChildElements(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<parent><a/><b/><c/></parent>\n`
	p := xml.NewParser(strings.NewReader(docSrc))
	parent, err := p.ParseElement()
	assert.Nil(t, err)
	assert.NotNil(t, parent)
	childs := parent.Elements()
	assert.Equal(t, len(childs), 3)
	assert.Equal(t, childs[0].Name(), "a")
	assert.Equal(t, childs[1].Name(), "b")
	assert.Equal(t, childs[2].Name(), "c")
}
