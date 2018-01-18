/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml_test

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/assert"
)

const (
	testNode     = "ortuman"
	testDomain   = "jackal.im"
	testResource = "my_resource"

	testBareJid = "ortuman@jackal.im"
	testFullJid = "ortuman@jackal.im/my_resource"
)

func TestBadJID(t *testing.T) {
	_, err := xml.NewJIDString("ortuman@", false)
	assert.NotNil(t, err)
	longStr := ""
	for i := 0; i < 1074; i++ {
		longStr += "a"
	}
	_, err2 := xml.NewJID(longStr, testDomain, testResource, false)
	assert.NotNil(t, err2)
	_, err3 := xml.NewJID(testNode, longStr, testResource, false)
	assert.NotNil(t, err3)
	_, err4 := xml.NewJID(testNode, testDomain, longStr, false)
	assert.NotNil(t, err4)
}

func TestNewJID(t *testing.T) {
	j, err := xml.NewJID(testNode, testDomain, testResource, false)
	assert.Nil(t, err)
	assert.Equal(t, j.Node(), testNode)
	assert.Equal(t, j.Domain(), testDomain)
	assert.Equal(t, j.Resource(), testResource)
}

func TestNewJIDString(t *testing.T) {
	j, err := xml.NewJIDString(testFullJid, false)
	assert.Nil(t, err)
	assert.Equal(t, j.Node(), testNode)
	assert.Equal(t, j.Domain(), testDomain)
	assert.Equal(t, j.Resource(), testResource)
	assert.Equal(t, j.String(), testFullJid)
}

func TestJIDEqual(t *testing.T) {
	j1, _ := xml.NewJIDString(testFullJid, false)
	j2, _ := xml.NewJID(testNode, testDomain, testResource, false)
	assert.NotNil(t, j1)
	assert.NotNil(t, j2)
	assert.Equal(t, j1.IsEqual(j2), true)
}
