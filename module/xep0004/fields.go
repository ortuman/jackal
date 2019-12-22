package xep0004

// Fields represent a set of form fields
type Fields []Field

// ValueForField returns the associated value for a given field name.
func (f Fields) ValueForField(fieldName string) string {
	return f.ValueForFieldOfType(fieldName, "")
}

// ValuesForField returns all associated values for a given field name.
func (f Fields) ValuesForField(fieldName string) []string {
	return f.ValuesForFieldOfType(fieldName, "")
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

// ValuesForFieldOfType returns all associated values for a given field name and type.
func (f Fields) ValuesForFieldOfType(fieldName, typ string) []string {
	var res []string
	for _, field := range f {
		if field.Var == fieldName && field.Type == typ && len(field.Values) > 0 {
			res = append(res, field.Values[0])
		}
	}
	return res
}
