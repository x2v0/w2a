package kotlin

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var serializedNameRe = regexp.MustCompile(`@SerializedName\(\s*"([^"]+)"\s*\)`)

func ExtractSerializedNames(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := make(map[string]struct{})
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		for _, m := range serializedNameRe.FindAllStringSubmatch(line, -1) {
			out[m[1]] = struct{}{}
		}
	}
	return out, sc.Err()
}
