package stack

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/x2v0/w2a/go/internal/apk"
)

// MergeManifest parses the bundle's manifest fragment (plain XML text OR
// binary AXML) and merges its contents into the target's AndroidManifest.xml.
//
// Supported merges:
//   - <uses-permission android:name="...">   (deduped)
//   - <uses-feature android:name="...">      (deduped)
//   - <queries><package/></queries>          (merged)
//   - <application> children:
//     <activity>, <service>, <receiver>, <provider>, <meta-data>,
//     <activity-alias>, <uses-library>       (appended)
//
// Returns the number of nodes appended.
func MergeManifest(target *apk.APK, b *Bundle) (int, error) {
	if len(b.Manifest) == 0 {
		return 0, nil
	}
	frag, err := parseManifestFragment(b.Manifest)
	if err != nil {
		return 0, fmt.Errorf("stack: parse manifest fragment: %w", err)
	}
	if frag == nil || (frag.Name != "manifest" && frag.Name != "fragment") {
		return 0, fmt.Errorf("stack: fragment root must be <manifest> or <fragment>, got <%s>", frag.Name)
	}
	if target.Manifest == nil || target.Manifest.Root == nil {
		return 0, fmt.Errorf("stack: target manifest missing")
	}

	n := 0
	root := target.Manifest.Root

	existingPerms := map[string]bool{}
	existingFeats := map[string]bool{}
	for _, child := range root.Children {
		name := child.AttrValue("name")
		switch child.Name {
		case "uses-permission":
			existingPerms[name] = true
		case "uses-feature":
			existingFeats[name] = true
		}
	}

	for _, child := range frag.Children {
		switch child.Name {
		case "uses-permission":
			name := child.AttrValue("name")
			if name == "" || existingPerms[name] {
				continue
			}
			root.Children = append(root.Children, cloneNode(child, root))
			existingPerms[name] = true
			n++

		case "uses-feature":
			name := child.AttrValue("name")
			if name == "" || existingFeats[name] {
				continue
			}
			root.Children = append(root.Children, cloneNode(child, root))
			existingFeats[name] = true
			n++

		case "queries":
			var q *apk.AXMLNode
			for _, c := range root.Children {
				if c.Name == "queries" {
					q = c
					break
				}
			}
			if q == nil {
				root.Children = append(root.Children, cloneNode(child, root))
				n++
				continue
			}
			// Merge children: <package> deduped by name; <intent> appended
			// (each intent is a distinct query pattern).
			existingPkgs := map[string]bool{}
			for _, sub := range q.Children {
				if sub.Name == "package" {
					existingPkgs[sub.AttrValue("name")] = true
				}
			}
			for _, sub := range child.Children {
				switch sub.Name {
				case "package":
					name := sub.AttrValue("name")
					if name == "" || existingPkgs[name] {
						continue
					}
					q.Children = append(q.Children, cloneNode(sub, q))
					existingPkgs[name] = true
					n++
				case "intent", "provider":
					q.Children = append(q.Children, cloneNode(sub, q))
					n++
				}
			}

		case "application":
			app := root.FindChild("application")
			if app == nil {
				continue
			}
			for _, ac := range child.Children {
				switch ac.Name {
				case "activity", "service", "receiver", "provider", "meta-data",
					"activity-alias", "uses-library":
					app.Children = append(app.Children, cloneNode(ac, app))
					n++
				}
			}
		}
	}
	return n, nil
}

// parseManifestFragment auto-detects plain XML vs binary AXML and returns
// an AXMLNode tree.
func parseManifestFragment(data []byte) (*apk.AXMLNode, error) {
	// Detect: AXML starts with 0x03 0x00 (chunkXML); plain XML starts with '<'.
	trim := bytes.TrimLeftFunc(data, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\uFEFF'
	})
	if len(trim) >= 2 && trim[0] == 0x03 && trim[1] == 0x00 {
		doc, err := apk.ParseAXML(data)
		if err != nil {
			return nil, err
		}
		return doc.Root, nil
	}
	// Plain text XML — parse with encoding/xml.
	return parsePlainXML(trim)
}

// xmlNode is a recursive struct for encoding/xml.
type xmlNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []xmlNode  `xml:",any"`
	Text     string     `xml:",chardata"`
}

// parsePlainXML parses text XML into AXMLNode.
func parsePlainXML(data []byte) (*apk.AXMLNode, error) {
	var root xmlNode
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("xml: %w", err)
	}
	return xmlToAXML(&root, nil), nil
}

func xmlToAXML(src *xmlNode, parent *apk.AXMLNode) *apk.AXMLNode {
	node := &apk.AXMLNode{
		Name:   src.XMLName.Local,
		Parent: parent,
		Text:   strings.TrimSpace(src.Text),
	}
	for _, a := range src.Attrs {
		ns := a.Name.Space
		name := a.Name.Local
		// Normalize "android" prefix to the AOSP namespace URI.
		if ns == "android" {
			ns = "http://schemas.android.com/apk/res/android"
		}
		node.Attributes = append(node.Attributes, apk.AXMLAttribute{
			Namespace: ns,
			Name:      name,
			RawValue:  a.Value,
			ValueStr:  a.Value,
			ValueType: apk.TypeString,
		})
	}
	for i := range src.Children {
		node.Children = append(node.Children, xmlToAXML(&src.Children[i], node))
	}
	return node
}

// cloneNode deep-copies a node (and children) with a new parent.
func cloneNode(src, parent *apk.AXMLNode) *apk.AXMLNode {
	n := &apk.AXMLNode{
		Name:       src.Name,
		Namespace:  src.Namespace,
		Attributes: append([]apk.AXMLAttribute(nil), src.Attributes...),
		Parent:     parent,
		Text:       src.Text,
	}
	for _, c := range src.Children {
		n.Children = append(n.Children, cloneNode(c, n))
	}
	return n
}
