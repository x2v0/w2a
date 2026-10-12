package apk

import (
	"os"
	"testing"
)

func TestFindIcons(t *testing.T) {
	apkPath := os.Getenv("WTA_TEST_APK")
	if apkPath == "" {
		t.Skip("no WTA_TEST_APK")
	}
	a, err := LoadAPK(apkPath)
	if err != nil {
		t.Fatal(err)
	}
	icons := a.FindIcons()
	t.Logf("found %d icon entries", len(icons))
	for i, ic := range icons {
		if i >= 10 {
			break
		}
		t.Logf("  %s (%d bytes) in %s", ic.Name, ic.Size, ic.Folder)
	}
}
