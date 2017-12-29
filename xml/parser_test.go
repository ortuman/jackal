/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
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
	p := xml.NewParser()
	a, err := p.ParseElement(strings.NewReader(docSrc))
	assert.Nil(t, err)
	assert.NotNil(t, a)
	assert.Equal(t, a.Name(), "a")
	assert.Equal(t, a.Attribute("xmlns"), "im.jackal")
	assert.Equal(t, a.Text(), "Hi!")
}

func TestFailedDocParse(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a><b><c a="attr1">HI</c><b></a>\n`
	p := xml.NewParser()
	_, err := p.ParseElement(strings.NewReader(docSrc))
	assert.NotNil(t, err)

	docSrc2 := `<?xml version="1.0" encoding="UTF-8"?>\n<element a="attr1">\n`
	p = xml.NewParser()
	element, err := p.ParseElement(strings.NewReader(docSrc2))
	assert.Equal(t, err, io.EOF)
	assert.Nil(t, element)
}

func TestDocChildElements(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<parent><a/><b/><c/></parent>\n`
	p := xml.NewParser()
	parent, err := p.ParseElement(strings.NewReader(docSrc))
	assert.Nil(t, err)
	assert.NotNil(t, parent)
	childs := parent.Elements()
	assert.Equal(t, len(childs), 3)
	assert.Equal(t, childs[0].Name(), "a")
	assert.Equal(t, childs[1].Name(), "b")
	assert.Equal(t, childs[2].Name(), "c")
}
