/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml_test

import (
	"testing"

	"github.com/ortuman/jackal/xml"
	"github.com/stretchr/testify/assert"
)

const (
	testJidNode     = "ortuman"
	testJidDomain   = "jackal.im"
	testJidResource = "my_resource"

	testBareJid = "ortuman@jackal.im"
	testFullJid = "ortuman@jackal.im/my_resource"
)

func TestNewJID(t *testing.T) {
	j, err := xml.NewJID(testJidNode, testJidDomain, testJidResource, false)
	assert.Nil(t, err)
	assert.Equal(t, j.Node, testJidNode)
	assert.Equal(t, j.Domain, testJidDomain)
	assert.Equal(t, j.Resource, testJidResource)
}

func TestNewJIDString(t *testing.T) {
	j, err := xml.NewJIDString(testFullJid, false)
	assert.Nil(t, err)
	assert.Equal(t, j.Node, testJidNode)
	assert.Equal(t, j.Domain, testJidDomain)
	assert.Equal(t, j.Resource, testJidResource)
	assert.Equal(t, j.ToBareJID(), testBareJid)
	assert.Equal(t, j.ToFullJID(), testFullJid)
	assert.Equal(t, j.String(), testFullJid)
}
