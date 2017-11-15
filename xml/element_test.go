/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

import "testing"

func TestElementNameAndNamespace(t *testing.T) {
	e := NewElementNamespace("iq", "jabber:client")
	if e.Name() != "iq" {
		t.Errorf("name %s. expected %s", e.Name(), "iq")
	}
	if e.Namespace() != "jabber:client" {
		t.Errorf("namespace %s. expected %s", e.Namespace(), "jabber:client")
	}
}

func TestShadowCopy(t *testing.T) {

}
