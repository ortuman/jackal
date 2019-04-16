package pubsubmodel

import (
	"encoding/gob"
)

type Option struct {
	Name  string
	Value string
}

type Node struct {
	Host    string
	Name    string
	Options []Option
}

// FromGob deserializes a User entity from it's gob binary representation.
func (n *Node) FromGob(dec *gob.Decoder) error {
	/*gobserializer.Decode(dec, &n.Host)
	gobserializer.Decode(dec, &n.Name)

	var optionsLen int
	gobserializer.Decode(dec, &optionsLen)
	for i := 0; i < optionsLen; i++ {
		var opt Option
		gobserializer.Decode(dec, &opt.Name)
		gobserializer.Decode(dec, &opt.Value)

		n.Options = append(n.Options, opt)
	}*/
	return nil
}

// ToGob converts a User entity to it's gob binary representation.
func (n *Node) ToGob(enc *gob.Encoder) {
	/*
		gobserializer.Encode(enc, n.Host)
		gobserializer.Encode(enc, n.Name)

		gobserializer.Encode(enc, len(n.Options))
		for _, opt := range n.Options {
			gobserializer.Encode(enc, opt.Name)
			gobserializer.Encode(enc, opt.Name)
		} */
}
