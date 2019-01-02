/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xmpp

// SetName sets XML node name.
func (e *Element) SetName(name string) *Element {
	e.name = name
	return e
}

// SetAttribute sets an XML node attribute (label=value)
func (e *Element) SetAttribute(label, value string) *Element {
	e.attrs.setAttribute(label, value)
	return e
}

// RemoveAttribute removes an XML node attribute.
func (e *Element) RemoveAttribute(label string) *Element {
	e.attrs.removeAttribute(label)
	return e
}

// SetNamespace sets 'xmlns' node attribute.
func (e *Element) SetNamespace(namespace string) *Element {
	e.attrs.setAttribute("xmlns", namespace)
	return e
}

// SetText sets XML node text value.
func (e *Element) SetText(text string) *Element {
	e.text = text
	return e
}

// SetID sets 'id' node attribute.
func (e *Element) SetID(identifier string) *Element {
	e.attrs.setAttribute("id", identifier)
	return e
}

// SetLanguage sets 'xml:lang' node attribute.
func (e *Element) SetLanguage(language string) *Element {
	e.attrs.setAttribute("xml:lang", language)
	return e
}

// SetFrom sets 'from' node attribute.
func (e *Element) SetFrom(from string) *Element {
	e.attrs.setAttribute("from", from)
	return e
}

// SetTo sets 'to' node attribute.
func (e *Element) SetTo(to string) *Element {
	e.attrs.setAttribute("to", to)
	return e
}

// SetType sets 'type' node attribute.
func (e *Element) SetType(tp string) *Element {
	e.attrs.setAttribute("type", tp)
	return e
}

// SetVersion sets 'version' node attribute.
func (e *Element) SetVersion(version string) *Element {
	e.attrs.setAttribute("version", version)
	return e
}

// AppendElement appends a new sub element.
func (e *Element) AppendElement(element XElement) *Element {
	e.elements.append(element)
	return e
}

// AppendElements appends an array of sub elements.
func (e *Element) AppendElements(elements []XElement) *Element {
	e.elements.append(elements...)
	return e
}

// RemoveElements removes all elements with a given name.
func (e *Element) RemoveElements(name string) *Element {
	e.elements.remove(name)
	return e
}

// RemoveElementsNamespace removes all elements with a given name and namespace.
func (e *Element) RemoveElementsNamespace(name, namespace string) *Element {
	e.elements.removeNamespace(name, namespace)
	return e
}

// ClearElements removes all elements.
func (e *Element) ClearElements() *Element {
	e.elements.clear()
	return e
}
