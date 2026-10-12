package apk

import "fmt"

// ListPermissions returns all android:name values of <uses-permission>.
func (a *APK) ListPermissions() []string {
	var out []string
	if a.Manifest == nil || a.Manifest.Root == nil {
		return nil
	}
	for _, child := range a.Manifest.Root.FindChildren("uses-permission") {
		name := child.AttrValue("name")
		if name == "" {
			// Fallback: sometimes attribute is namespace-qualified.
			for i := range child.Attributes {
				if child.Attributes[i].Name == "name" {
					name = child.Attributes[i].ValueStr
					if name == "" {
						name = child.Attributes[i].RawValue
					}
				}
			}
		}
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// AddPermission appends a new <uses-permission android:name="..."/> node.
// Idempotent — returns error if already present.
func (a *APK) AddPermission(name string) error {
	if name == "" {
		return fmt.Errorf("apk: empty permission name")
	}
	if a.Manifest == nil || a.Manifest.Root == nil {
		return fmt.Errorf("apk: no manifest")
	}
	for _, existing := range a.ListPermissions() {
		if existing == name {
			return fmt.Errorf("apk: permission already present: %s", name)
		}
	}
	node := &AXMLNode{
		Name: "uses-permission",
		Attributes: []AXMLAttribute{
			{
				Name:      "name",
				Namespace: "http://schemas.android.com/apk/res/android",
				ValueType: TypeString,
				RawValue:  name,
				ValueStr:  name,
			},
		},
		Parent: a.Manifest.Root,
	}
	a.Manifest.Root.Children = append(a.Manifest.Root.Children, node)
	return nil
}

// RemovePermission deletes all <uses-permission> entries with the given name.
// Returns number removed.
func (a *APK) RemovePermission(name string) int {
	if a.Manifest == nil || a.Manifest.Root == nil {
		return 0
	}
	var kept []*AXMLNode
	removed := 0
	for _, child := range a.Manifest.Root.Children {
		if child.Name == "uses-permission" && child.AttrValue("name") == name {
			removed++
			continue
		}
		kept = append(kept, child)
	}
	a.Manifest.Root.Children = kept
	return removed
}
