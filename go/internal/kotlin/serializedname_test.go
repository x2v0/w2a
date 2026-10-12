package kotlin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSerializedNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Sample.kt")
	content := "package com.example\n" +
		"import com.google.gson.annotations.SerializedName\n" +
		"data class Config(\n" +
		"    @SerializedName(\"app_name\") val appName: String,\n" +
		"    // @SerializedName(\"commented_out\") - ignored\n" +
		"    @SerializedName(\"icon_url\") val iconUrl: String?,\n" +
		")\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ExtractSerializedNames(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["app_name"]; !ok {
		t.Errorf("missing app_name")
	}
	if _, ok := got["icon_url"]; !ok {
		t.Errorf("missing icon_url")
	}
	if _, ok := got["commented_out"]; ok {
		t.Errorf("commented_out should be ignored")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(got), got)
	}
}
