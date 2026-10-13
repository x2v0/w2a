package musl

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type hunkLine struct {
	kind byte
	text string
}

type hunk struct {
	oldStart int
	lines    []hunkLine
}

var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+\d+(?:,\d+)? @@`)

func ApplyPatch(root string, patchData []byte, strip int) error {
	sc := bufio.NewScanner(strings.NewReader(string(patchData)))
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var (
		path  string
		hunks []hunk
	)

	flush := func() error {
		if path == "" || len(hunks) == 0 {
			path, hunks = "", nil
			return nil
		}
		full := filepath.Join(root, stripPath(path, strip))
		err := applyHunks(full, hunks)
		path, hunks = "", nil
		return err
	}

	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "--- "):
			if err := flush(); err != nil {
				return err
			}
		case strings.HasPrefix(line, "+++ "):
			p := strings.TrimSpace(line[4:])
			if i := strings.IndexByte(p, '\t'); i >= 0 {
				p = p[:i]
			}
			path = p
		case hunkHeaderRe.MatchString(line):
			m := hunkHeaderRe.FindStringSubmatch(line)
			start, _ := strconv.Atoi(m[1])
			hunks = append(hunks, hunk{oldStart: start})
		case len(hunks) > 0 && len(line) > 0:
			k := line[0]
			if k == ' ' || k == '-' || k == '+' {
				hunks[len(hunks)-1].lines = append(hunks[len(hunks)-1].lines,
					hunkLine{kind: k, text: line[1:]})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return flush()
}

func stripPath(p string, n int) string {
	if n <= 0 {
		return p
	}
	parts := strings.Split(p, "/")
	if len(parts) <= n {
		return p
	}
	return strings.Join(parts[n:], "/")
}

func applyHunks(path string, hunks []hunk) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	orig := strings.Split(string(data), "\n")
	hadNL := len(data) > 0 && data[len(data)-1] == '\n'
	if hadNL {
		orig = orig[:len(orig)-1]
	}

	var out []string
	pos := 0
	for _, h := range hunks {
		target := h.oldStart - 1
		if target < pos {
			return fmt.Errorf("overlapping hunk at line %d", target+1)
		}
		out = append(out, orig[pos:target]...)
		pos = target
		for _, hl := range h.lines {
			switch hl.kind {
			case ' ':
				if pos >= len(orig) || orig[pos] != hl.text {
					return fmt.Errorf("context mismatch line %d", pos+1)
				}
				out = append(out, orig[pos])
				pos++
			case '-':
				if pos >= len(orig) || orig[pos] != hl.text {
					return fmt.Errorf("removed mismatch line %d", pos+1)
				}
				pos++
			case '+':
				out = append(out, hl.text)
			}
		}
	}
	out = append(out, orig[pos:]...)

	result := strings.Join(out, "\n")
	if hadNL {
		result += "\n"
	}
	return os.WriteFile(path, []byte(result), 0o644)
}
