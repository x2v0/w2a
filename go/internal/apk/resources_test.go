package apk

import (
	"os"
	"testing"
)

func TestLoadResourcesFromAPK(t *testing.T) {
	apkPath := os.Getenv("WTA_TEST_APK")
	if apkPath == "" {
		t.Skip("no WTA_TEST_APK")
	}
	a, err := LoadAPK(apkPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("package: %s", a.Package())

	table, err := a.LoadResources()
	if err != nil {
		t.Skipf("no resources.arsc: %v", err)
	}
	t.Logf("global pool: %d strings", len(table.GlobalPool.Strings))
	t.Logf("packages: %d", len(table.Packages))
	for _, pkg := range table.Packages {
		t.Logf("  package 0x%02X %s (%d types)",
			pkg.ID, pkg.Name, len(pkg.Types))
	}

	// Resolve a known string.
	if len(table.GlobalPool.Strings) > 0 {
		t.Logf("  string[0] = %q", table.GlobalPool.Strings[0])
	}
}

func TestSaveResourcesRoundtrip(t *testing.T) {
	apkPath := os.Getenv("WTA_TEST_APK")
	if apkPath == "" {
		t.Skip("no WTA_TEST_APK")
	}
	a, err := LoadAPK(apkPath)
	if err != nil {
		t.Fatal(err)
	}
	table, err := a.LoadResources()
	if err != nil {
		t.Skipf("no resources: %v", err)
	}
	before := len(a.Archive.Get("resources.arsc").Data)
	t.Logf("resources.arsc before: %d bytes", before)

	if err := a.SaveResources(table); err != nil {
		t.Fatal(err)
	}
	after := len(a.Archive.Get("resources.arsc").Data)
	t.Logf("resources.arsc after:  %d bytes", after)

	// Verify we can re-load it.
	table2, err := a.LoadResources()
	if err != nil {
		t.Fatalf("reload after save: %v", err)
	}
	if len(table2.GlobalPool.Strings) != len(table.GlobalPool.Strings) {
		t.Errorf("pool size mismatch: %d vs %d",
			len(table2.GlobalPool.Strings), len(table.GlobalPool.Strings))
	}
}
