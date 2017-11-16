/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml_test

import (
	"testing"

	"github.com/ortuman/jackal/xml"
)

func TestSetName(t *testing.T) {
	m := xml.NewMutableElementName("n")
	m.SetName("new_n")
	if m.Name() != "new_n" {
		t.Fatal(`m.Name() != "new_n"`)
	}
}

func TestSetText(t *testing.T) {
	m := xml.NewMutableElementName("n")
	m.SetText("example text")
	if m.Text() != "example text" {
		t.Fatal(`m.Text() != "example text"`)
	}
}

func TestMutableCopy(t *testing.T) {
}
