package modules

import (
	"os/exec"
	"strings"
)

// Git runs a git command in repoRoot and returns stdout.
func Git(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	return string(out), err
}

// LastCommitFor returns the most recent commit SHA that touched path,
// or "" if git fails.
func LastCommitFor(repoRoot, relPath string) string {
	out, err := Git(repoRoot, "log", "-n", "1", "--format=%H", "--", relPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// CommitMetadata returns author name, email, ISO timestamp, and subject.
type CommitMeta struct {
	AuthorName  string
	AuthorEmail string
	AuthorDate  string
	Subject     string
}

func GetCommitMeta(repoRoot, sha string) CommitMeta {
	out, err := Git(repoRoot, "show", "-s", "--format=%an%n%ae%n%aI%n%s", sha)
	if err != nil {
		return CommitMeta{}
	}
	lines := strings.Split(out, "\n")
	m := CommitMeta{}
	if len(lines) > 0 {
		m.AuthorName = lines[0]
	}
	if len(lines) > 1 {
		m.AuthorEmail = lines[1]
	}
	if len(lines) > 2 {
		m.AuthorDate = lines[2]
	}
	if len(lines) > 3 {
		m.Subject = lines[3]
	}
	return m
}

// AllCommitsFor returns every commit SHA that touched relPath, newest first.
func AllCommitsFor(repoRoot, relPath string) []string {
	out, err := Git(repoRoot, "log", "--format=%H", "--", relPath)
	if err != nil {
		return nil
	}
	var shas []string
	for _, line := range strings.Split(out, "\n") {
		s := strings.TrimSpace(line)
		if s != "" {
			shas = append(shas, s)
		}
	}
	return shas
}

// IntroducingCommit returns the oldest commit that added relPath/module.json,
// or "" if not found.
func IntroducingCommit(repoRoot, relPath string) string {
	out, err := Git(repoRoot, "log", "--diff-filter=A", "--format=%H", "--", relPath+"/module.json")
	if err != nil {
		return ""
	}
	lines := []string{}
	for _, line := range strings.Split(out, "\n") {
		s := strings.TrimSpace(line)
		if s != "" {
			lines = append(lines, s)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}

// PRNumbersFromText extracts "#123" PR references from arbitrary text.
func PRNumbersFromText(texts ...string) []int {
	seen := map[int]struct{}{}
	var out []int
	for _, t := range texts {
		if t == "" {
			continue
		}
		for _, m := range PRRefRE.FindAllStringSubmatch(t, -1) {
			var n int
			for _, ch := range m[1] {
				n = n*10 + int(ch-'0')
			}
			if _, dup := seen[n]; !dup {
				seen[n] = struct{}{}
				out = append(out, n)
			}
		}
	}
	return out
}
