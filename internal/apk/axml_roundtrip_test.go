package apk

import "testing"

func TestRoundtripSimple(t *testing.T) {
	root := &AXMLNode{
		Name: "manifest",
		Attributes: []AXMLAttribute{
			{Name: "package", ValueType: TypeString, RawValue: "com.example.app", ValueStr: "com.example.app"},
		},
	}
	app := &AXMLNode{
		Name:   "application",
		Parent: root,
		Attributes: []AXMLAttribute{
			{Name: "label", ValueType: TypeString, RawValue: "Test App", ValueStr: "Test App"},
		},
	}
	root.Children = []*AXMLNode{app}

	doc := &AXMLDocument{Root: root}
	data, err := WriteAXML(doc)
	if err != nil {
		t.Fatalf("WriteAXML: %v", err)
	}
	if len(data) < 100 {
		t.Fatalf("suspiciously small output: %d bytes", len(data))
	}
	t.Logf("wrote %d bytes", len(data))

	doc2, err := ParseAXML(data)
	if err != nil {
		t.Fatalf("ParseAXML: %v", err)
	}
	if doc2.Root == nil {
		t.Fatal("nil root after parse")
	}
	if doc2.Root.Name != "manifest" {
		t.Errorf("root name: got %q, want manifest", doc2.Root.Name)
	}
	if got := doc2.Root.AttrValue("package"); got != "com.example.app" {
		t.Errorf("package: got %q", got)
	}
	app2 := doc2.Root.FindChild("application")
	if app2 == nil {
		t.Fatal("application missing after roundtrip")
	}
	if got := app2.AttrValue("label"); got != "Test App" {
		t.Errorf("label: got %q", got)
	}
}

func TestRoundtripModify(t *testing.T) {
	root := &AXMLNode{
		Name: "manifest",
		Attributes: []AXMLAttribute{
			{Name: "package", ValueType: TypeString, RawValue: "com.old.pkg", ValueStr: "com.old.pkg"},
		},
	}
	doc := &AXMLDocument{Root: root}
	root.SetAttr("package", "com.new.pkg")
	root.SetAttr("versionName", "2.0.0")

	data, err := WriteAXML(doc)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := ParseAXML(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc2.Root.AttrValue("package"); got != "com.new.pkg" {
		t.Errorf("package: got %q", got)
	}
	if got := doc2.Root.AttrValue("versionName"); got != "2.0.0" {
		t.Errorf("versionName: got %q", got)
	}
}
