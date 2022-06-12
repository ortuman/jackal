// Copyright 2022 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xep0004

import "github.com/jackal-xmpp/stravaganza"

const (
	// StringDataType datatype represents character strings in XML.
	StringDataType = "xs:string"

	// BooleanDataType represents the values of two-valued logic.
	BooleanDataType = "xs:boolean"

	// DecimalDataType represents a subset of the real numbers, which can be represented by decimal numerals.
	DecimalDataType = "xs:decimal"

	// FloatDataType is patterned after the IEEE single-precision 32-bit floating point datatype
	FloatDataType = "xs:float"

	// DoubleDataType is patterned after the IEEE double-precision 64-bit floating point datatype.
	DoubleDataType = "xs:double"

	// DurationDataType is a datatype that represents durations of time.
	DurationDataType = "xs:duration"

	// DateTimeDataType represents instants of time, optionally marked with a particular time zone offset.
	DateTimeDataType = "xs:dateTime"

	// HexBinaryDataType represents arbitrary hex-encoded binary data.
	HexBinaryDataType = "xs:hexBinary"

	// Base64BinaryDataType represents arbitrary Base64-encoded binary data
	Base64BinaryDataType = "xs:base64Binary"
)

const validateNamespace = "http://jabber.org/protocol/xdata-validate"

// Validator defines validation type interface.
type Validator interface {
	Element() stravaganza.Element
}

// Validate represents a field validation type.
type Validate struct {
	DataType  string
	Validator Validator
}

// Element returns validation type element representation.
func (v *Validate) Element() stravaganza.Element {
	b := stravaganza.NewBuilder("validate").
		WithAttribute(stravaganza.Namespace, validateNamespace).
		WithAttribute("datatype", v.DataType)
	if v.Validator != nil {
		b.WithChild(v.Validator.Element())
	}
	return b.Build()
}

// OpenValidator represents open validation type.
type OpenValidator struct{}

// Element satisfies Validator interface.
func (v *OpenValidator) Element() stravaganza.Element {
	return stravaganza.NewBuilder("open").Build()
}

// BasicValidator represents basic validation type.
type BasicValidator struct{}

// Element satisfies Validator interface.
func (v *BasicValidator) Element() stravaganza.Element {
	return stravaganza.NewBuilder("basic").Build()
}

// RangeValidator represents range validation type.
type RangeValidator struct {
	Min string
	Max string
}

// Element satisfies Validator interface.
func (v *RangeValidator) Element() stravaganza.Element {
	b := stravaganza.NewBuilder("range")
	if len(v.Min) > 0 {
		b.WithAttribute("min", v.Min)
	}
	if len(v.Max) > 0 {
		b.WithAttribute("max", v.Max)
	}
	return b.Build()
}

// RegExValidator represents regex validation type.
type RegExValidator struct {
	RegEx string
}

// Element satisfies Validator interface.
func (v *RegExValidator) Element() stravaganza.Element {
	b := stravaganza.NewBuilder("regex")
	b.WithText(v.RegEx)
	return b.Build()
}
