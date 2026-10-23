package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStarterYAML_AllTypes(t *testing.T) {
	cases := []struct{ typ, wantType string }{
		{"chat", "type: html"}, {"notes", "type: html"},
		{"dashboard", "type: html"}, {"rss", "type: html"}, {"reader", "type: html"},
		{"html", "type: html"}, {"web", "type: web"}, {"frontend", "type: frontend"},
		{"go", "type: go"}, {"gallery", "type: gallery"}, {"multiweb", "type: multiweb"},
	}
	for _, c := range cases {
		yaml := starterYAML(c.typ, "Test", "com.test", "https://example.com")
		if !strings.Contains(yaml, c.wantType) {
			t.Errorf("%s: want %q", c.typ, c.wantType)
		}
		if !strings.Contains(yaml, `name: "Test"`) || !strings.Contains(yaml, "package: com.test") {
			t.Errorf("%s: missing name/package", c.typ)
		}
	}
}

func TestStarterYAML_HtmlHasProject(t *testing.T) {
	y := starterYAML("chat", "X", "com.x", "")
	if !strings.Contains(y, "source: ./html-project") {
		t.Error("chat: missing html-project source")
	}
}

func TestScaffoldAll_CreatesFiles(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "wta-")
	defer os.RemoveAll(tmp)
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(tmp)
	fns := map[string]func(string) error{
		"chat": scaffoldChat, "notes": scaffoldNotes,
		"dashboard": scaffoldDashboard, "rss": scaffoldRss, "reader": scaffoldReader,
	}
	for n, fn := range fns {
		os.RemoveAll(htmlProjectDir)
		if err := fn("T"); err != nil {
			t.Errorf("%s: %v", n, err); continue
		}
		if _, err := os.Stat(filepath.Join(htmlProjectDir, "index.html")); err != nil {
			t.Errorf("%s: index.html", n)
		}
		c, _ := os.ReadFile(filepath.Join(htmlProjectDir, "mtapi-debug.js"))
		if !strings.Contains(string(c), "MTAPI_PERF_V3") {
			t.Errorf("%s: debug.js missing PERF_V3", n)
		}
		if !strings.Contains(string(c), "MTAPI_LOGCAT_REVERSE") {
			t.Errorf("%s: debug.js missing LOGCAT_REVERSE", n)
		}
	}
}

func TestScaffoldChat_Content(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "wta-")
	defer os.RemoveAll(tmp)
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(tmp)
	scaffoldChat("T")
	s, _ := os.ReadFile(filepath.Join(htmlProjectDir, "index.html"))
	for _, m := range []string{"MtapiDebug.attach", "chat-cfg"} {
		if !strings.Contains(strings.ToLower(string(s)), strings.ToLower(m)) {
			t.Errorf("chat: %q", m)
		}
	}
}

func TestScaffoldNotes_LocalStorage(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "wta-")
	defer os.RemoveAll(tmp)
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(tmp)
	scaffoldNotes("T")
	s, _ := os.ReadFile(filepath.Join(htmlProjectDir, "index.html"))
	if !strings.Contains(string(s), "localStorage") {
		t.Error("notes: no localStorage")
	}
}

func TestScaffoldDashboard_Fetch(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "wta-")
	defer os.RemoveAll(tmp)
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(tmp)
	scaffoldDashboard("T")
	s, _ := os.ReadFile(filepath.Join(htmlProjectDir, "index.html"))
	for _, m := range []string{"fetch(", "renderTable"} {
		if !strings.Contains(string(s), m) {
			t.Errorf("dashboard: %q", m)
		}
	}
}

func TestScaffoldRss_DOMParser(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "wta-")
	defer os.RemoveAll(tmp)
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(tmp)
	scaffoldRss("T")
	s, _ := os.ReadFile(filepath.Join(htmlProjectDir, "index.html"))
	for _, m := range []string{"DOMParser", "text/xml"} {
		if !strings.Contains(string(s), m) {
			t.Errorf("rss: %q", m)
		}
	}
}

func TestScaffoldReader_Md2Html(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "wta-")
	defer os.RemoveAll(tmp)
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir(tmp)
	scaffoldReader("T")
	s, _ := os.ReadFile(filepath.Join(htmlProjectDir, "index.html"))
	for _, m := range []string{"md2html", `type="file"`} {
		if !strings.Contains(string(s), m) {
			t.Errorf("reader: %q", m)
		}
	}
}
