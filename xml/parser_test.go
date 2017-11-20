/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml_test

import (
	"strings"
	"testing"

	"github.com/ortuman/jackal/xml"
)

func TestDocParse(t *testing.T) {
	docSrc := `<?xml version="1.0" encoding="UTF-8"?>\n<a/>\n`
	p := xml.NewParser()
	p.ParseElements(strings.NewReader(docSrc))
}
