/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ortuman/jackal/xml"
)

func TestDocParse(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a xmlns="im.jackal">Hi!</a>\n`
	p := xml.NewParser()
	err := p.ParseElements(strings.NewReader(docSrc))
	assert.Nil(t, err)
	a := p.PopElement()
	assert.NotNil(t, a)
	assert.Equal(t, a.Name(), "a")
	assert.Equal(t, a.Attribute("xmlns"), "im.jackal")
	assert.Equal(t, a.Text(), "Hi!")
}

func TestFailedDocParse(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a><b><c a="attr1">HI</c><b></a>\n`
	p := xml.NewParser()
	err := p.ParseElements(strings.NewReader(docSrc))
	assert.NotNil(t, err)
}

func TestDocChildElements(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<parent><a/><b/><c/></parent>\n`
	p := xml.NewParser()
	err := p.ParseElements(strings.NewReader(docSrc))
	assert.Nil(t, err)
	parent := p.PopElement()
	assert.NotNil(t, parent)
	childs := parent.Elements()
	assert.Equal(t, len(childs), 3)
	assert.Equal(t, childs[0].Name(), "a")
	assert.Equal(t, childs[1].Name(), "b")
	assert.Equal(t, childs[2].Name(), "c")
}
