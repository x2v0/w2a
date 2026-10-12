package apk

import "strings"

// FindChild returns the first direct child with the given (case-sensitive) name.
func (n *AXMLNode) FindChild(name string) *AXMLNode {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// FindChildren returns all direct children matching name.
func (n *AXMLNode) FindChildren(name string) []*AXMLNode {
	if n == nil {
		return nil
	}
	var out []*AXMLNode
	for _, c := range n.Children {
		if c.Name == name {
			out = append(out, c)
		}
	}
	return out
}

// FindDescendant does a DFS search for the first node matching name.
func (n *AXMLNode) FindDescendant(name string) *AXMLNode {
	if n == nil {
		return nil
	}
	if n.Name == name {
		return n
	}
	for _, c := range n.Children {
		if found := c.FindDescendant(name); found != nil {
			return found
		}
	}
	return nil
}

// Attr returns the attribute with the given name (ignoring namespace).
// Returns nil if not found.
func (n *AXMLNode) Attr(name string) *AXMLAttribute {
	if n == nil {
		return nil
	}
	for i := range n.Attributes {
		if n.Attributes[i].Name == name {
			return &n.Attributes[i]
		}
	}
	return nil
}

// AttrValue returns the string value of an attribute, or "" if not found.
// If the attribute is a string reference, returns the resolved string;
// otherwise returns RawValue.
func (n *AXMLNode) AttrValue(name string) string {
	a := n.Attr(name)
	if a == nil {
		return ""
	}
	if a.ValueStr != "" {
		return a.ValueStr
	}
	return a.RawValue
}

// SetAttr adds or replaces an attribute by name. Uses string value type.
func (n *AXMLNode) SetAttr(name, value string) {
	if n == nil {
		return
	}
	for i := range n.Attributes {
		if n.Attributes[i].Name == name {
			n.Attributes[i].RawValue = value
			n.Attributes[i].ValueStr = value
			n.Attributes[i].ValueType = TypeString
			return
		}
	}
	n.Attributes = append(n.Attributes, AXMLAttribute{
		Name:      name,
		RawValue:  value,
		ValueStr:  value,
		ValueType: TypeString,
	})
}

// HasAttr reports whether the node has an attribute with the given name.
func (n *AXMLNode) HasAttr(name string) bool {
	return n.Attr(name) != nil
}

// Path returns a slash-separated path from root to this node, for debugging.
func (n *AXMLNode) Path() string {
	if n == nil {
		return ""
	}
	var parts []string
	cur := n
	for cur != nil {
		parts = append([]string{cur.Name}, parts...)
		cur = cur.Parent
	}
	return strings.Join(parts, "/")
}
