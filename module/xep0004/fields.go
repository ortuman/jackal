package xep0004

import (
	"strconv"
	"strings"
)

// Fields represent a set of form fields
type Fields []Field

// BoolForField returns the associated boolean value for a given field.
func (f Fields) BoolForField(fieldName string) bool {
	return f.BoolForFieldOfType(fieldName, "")
}

// BoolForFieldOfType returns the associated boolean value for a given field and type.
func (f Fields) BoolForFieldOfType(fieldName, typ string) bool {
	v := f.ValueForFieldOfType(fieldName, typ)
	return v == "1" || strings.ToLower(v) == "true"
}

// IntForField returns the associated integer value for a given field.
func (f Fields) IntForField(fieldName string) int {
	return f.IntForFieldOfType(fieldName, "")
}

// IntForFieldOfType returns the associated integer value for a given field and type.
func (f Fields) IntForFieldOfType(fieldName, typ string) int {
	i, _ := strconv.Atoi(f.ValueForFieldOfType(fieldName, typ))
	return i
}

// ValueForField returns the associated value for a given field name.
func (f Fields) ValueForField(fieldName string) string {
	return f.ValueForFieldOfType(fieldName, "")
}

// ValueForFieldOfType returns the associated value for a given field name and type.
func (f Fields) ValueForFieldOfType(fieldName, typ string) string {
	for _, field := range f {
		if field.Var == fieldName && field.Type == typ && len(field.Values) > 0 {
			return field.Values[0]
		}
	}
	return ""
}
