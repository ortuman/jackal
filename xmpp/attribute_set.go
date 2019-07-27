/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

import (
	"bytes"
	"encoding/gob"
)

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	Label string
	Value string
}

// AttributeSet interface represents a read-only set of XML attributes.
type AttributeSet interface {
	Get(string) string
	Count() int
}

type attributeSet []Attribute

func (as attributeSet) Get(label string) string {
	for _, attr := range as {
		if attr.Label == label {
			return attr.Value
		}
	}
	return ""
}

func (as attributeSet) Count() int {
	return len(as)
}

func (as *attributeSet) setAttribute(label, value string) {
	for i := 0; i < len(*as); i++ {
		if (*as)[i].Label == label {
			(*as)[i].Value = value
			return
		}
	}
	*as = append(*as, Attribute{label, value})
}

func (as *attributeSet) removeAttribute(label string) {
	for i := 0; i < len(*as); i++ {
		if (*as)[i].Label == label {
			*as = append((*as)[:i], (*as)[i+1:]...)
			return
		}
	}
}

func (as *attributeSet) copyFrom(from attributeSet) {
	*as = make([]Attribute, from.Count())
	copy(*as, from)
}

func (as *attributeSet) FromBytes(buf *bytes.Buffer) error {
	dec := gob.NewDecoder(buf)
	var c int
	if err := dec.Decode(&c); err != nil {
		return err
	}
	for i := 0; i < c; i++ {
		var attr Attribute
		if err := dec.Decode(&attr.Label); err != nil {
			return err
		}
		if err := dec.Decode(&attr.Value); err != nil {
			return err
		}
		*as = append(*as, attr)
	}
	return nil
}

func (as attributeSet) ToBytes(buf *bytes.Buffer) error {
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(len(as)); err != nil {
		return err
	}
	for _, attr := range as {
		if err := enc.Encode(&attr.Label); err != nil {
			return err
		}
		if err := enc.Encode(&attr.Value); err != nil {
			return err
		}
	}
	return nil
}
