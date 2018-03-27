/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xml

import (
	"encoding/gob"
	"io"
)

// Attribute represents an XML node attribute (label=value).
type Attribute struct {
	Label string
	Value string
}

type AttributeSet struct {
	attrs []Attribute
}

func (as *AttributeSet) Get(label string) string {
	for _, attr := range as.attrs {
		if attr.Label == label {
			return attr.Value
		}
	}
	return ""
}

func (as *AttributeSet) Len() int {
	return len(as.attrs)
}

func (as *AttributeSet) setAttribute(label, value string) {
	for i := 0; i < len(as.attrs); i++ {
		if as.attrs[i].Label == label {
			as.attrs[i].Value = value
			return
		}
	}
	as.attrs = append(as.attrs, Attribute{label, value})
}

func (as *AttributeSet) removeAttribute(label string) {
	for i := 0; i < len(as.attrs); i++ {
		if as.attrs[i].Label == label {
			as.attrs = append(as.attrs[:i], as.attrs[i+1:]...)
			return
		}
	}
}

func (as *AttributeSet) copyFrom(from *AttributeSet) {
	as.attrs = make([]Attribute, from.Len())
	copy(as.attrs, from.attrs)
}

func (as *AttributeSet) toXML(w io.Writer) {
	for i := 0; i < len(as.attrs); i++ {
		if len(as.attrs[i].Value) == 0 {
			continue
		}
		w.Write([]byte(" "))
		w.Write([]byte(as.attrs[i].Label))
		w.Write([]byte(`="`))
		w.Write([]byte(as.attrs[i].Value))
		w.Write([]byte(`"`))
	}
}

func (as *AttributeSet) fromGob(dec *gob.Decoder) {
	var c int
	dec.Decode(&c)
	for i := 0; i < c; i++ {
		var attr Attribute
		dec.Decode(&attr.Label)
		dec.Decode(&attr.Value)
		as.attrs = append(as.attrs, attr)
	}
}

func (as *AttributeSet) toGob(enc *gob.Encoder) {
	enc.Encode(len(as.attrs))
	for _, attr := range as.attrs {
		enc.Encode(&attr.Label)
		enc.Encode(&attr.Value)
	}
}
