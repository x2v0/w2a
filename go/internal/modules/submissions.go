package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Contributor is a secondary contributor to a module.
type Contributor struct {
	Login   string `json:"login,omitempty"`
	Name    string `json:"name,omitempty"`
	Avatar  string `json:"avatarUrl,omitempty"`
	Profile string `json:"profileUrl,omitempty"`
}

// Submission is one entry in submissions.json.
type Submission struct {
	PRNumber        *int          `json:"prNumber,omitempty"`
	PRURL           string        `json:"prUrl,omitempty"`
	SubmittedAt     string        `json:"submittedAt,omitempty"`
	Direct          bool          `json:"direct,omitempty"`
	SubmitterLogin  string        `json:"-"`
	SubmitterName   string        `json:"-"`
	SubmitterAvatar string        `json:"-"`
	SubmitterProf   string        `json:"-"`
	Contributors    []Contributor `json:"contributors,omitempty"`
}

// ToJSON serialises a Submission for the output file.
func (s *Submission) ToJSON() map[string]any {
	out := map[string]any{}
	if s.PRNumber != nil {
		out["prNumber"] = *s.PRNumber
	}
	if s.PRURL != "" {
		out["prUrl"] = s.PRURL
	}
	if s.SubmittedAt != "" {
		out["submittedAt"] = s.SubmittedAt
	}
	if s.Direct {
		out["direct"] = true
	}
	submitter := map[string]any{}
	if s.SubmitterLogin != "" {
		submitter["login"] = s.SubmitterLogin
	}
	if s.SubmitterName != "" {
		submitter["name"] = s.SubmitterName
	}
	if s.SubmitterAvatar != "" {
		submitter["avatarUrl"] = s.SubmitterAvatar
	}
	if s.SubmitterProf != "" {
		submitter["profileUrl"] = s.SubmitterProf
	}
	if len(submitter) > 0 {
		out["submitter"] = submitter
	}
	if len(s.Contributors) > 0 {
		contribs := make([]map[string]any, 0, len(s.Contributors))
		for _, c := range s.Contributors {
			if c.Login == "" {
				continue
			}
			m := map[string]any{"login": c.Login}
			if c.Name != "" {
				m["name"] = c.Name
			}
			if c.Avatar != "" {
				m["avatarUrl"] = c.Avatar
			}
			if c.Profile != "" {
				m["profileUrl"] = c.Profile
			}
			contribs = append(contribs, m)
		}
		if len(contribs) > 0 {
			out["contributors"] = contribs
		}
	}
	return out
}

// UserCache caches resolved GitHub users across a run.
type UserCache struct {
	Users map[string]map[string]string
}

func NewUserCache() *UserCache {
	return &UserCache{Users: map[string]map[string]string{}}
}

func (c *UserCache) Resolve(client *GitHubClient, login string) map[string]string {
	if login == "" {
		return nil
	}
	if cached, ok := c.Users[login]; ok {
		return cached
	}
	resolved := client.User(login)
	entry := map[string]string{
		"name":    login,
		"avatar":  "",
		"profile": "https://github.com/" + login,
	}
	if resolved != nil {
		if n, ok := resolved["name"].(string); ok && n != "" {
			entry["name"] = n
		}
		if a, ok := resolved["avatar_url"].(string); ok {
			entry["avatar"] = a
		}
		if p, ok := resolved["html_url"].(string); ok && p != "" {
			entry["profile"] = p
		}
	}
	c.Users[login] = entry
	return entry
}

// ResolveForPath builds a Submission for a module folder.
func ResolveForPath(
	repoRoot, moduleRelPath string,
	client *GitHubClient,
	maintainers map[string]struct{},
	cache *UserCache,
) *Submission {
	sha := LastCommitFor(repoRoot, moduleRelPath)
	if sha == "" {
		return nil
	}

	intro := IntroducingCommit(repoRoot, moduleRelPath)
	targetSHA := sha
	if intro != "" {
		targetSHA = intro
	}

	meta := GetCommitMeta(repoRoot, targetSHA)

	if client == nil {
		if len(maintainers) == 0 {
			return nil
		}
		return &Submission{
			Direct:         true,
			SubmittedAt:    meta.AuthorDate,
			SubmitterLogin: meta.AuthorName,
			SubmitterName:  meta.AuthorName,
		}
	}

	pulls := client.PullsForCommit(targetSHA)
	if len(pulls) == 0 {
		for _, n := range PRNumbersFromText(meta.Subject) {
			if pr := client.GetPull(n); pr != nil {
				pulls = append(pulls, pr)
			}
		}
	}

	var merged []map[string]any
	for _, p := range pulls {
		if ma, ok := p["merged_at"].(string); ok && ma != "" {
			merged = append(merged, p)
		}
	}
	var onMain []map[string]any
	for _, p := range merged {
		base, _ := p["base"].(map[string]any)
		ref, _ := base["ref"].(string)
		if ref == "" || ref == "main" {
			onMain = append(onMain, p)
		}
	}
	if len(onMain) > 0 {
		merged = onMain
	}

	if len(merged) > 0 {
		sort.Slice(merged, func(i, j int) bool {
			a, _ := merged[i]["merged_at"].(string)
			b, _ := merged[j]["merged_at"].(string)
			return a < b
		})
		pr := merged[0]
		user, _ := pr["user"].(map[string]any)
		login, _ := user["login"].(string)
		avatar, _ := user["avatar_url"].(string)
		profile, _ := user["html_url"].(string)
		if profile == "" && login != "" {
			profile = "https://github.com/" + login
		}
		displayName := ""
		if login != "" {
			resolved := cache.Resolve(client, login)
			displayName = resolved["name"]
			if resolved["avatar"] != "" {
				avatar = resolved["avatar"]
			}
		}
		var prNumber *int
		if n, ok := pr["number"].(float64); ok {
			x := int(n)
			prNumber = &x
		}
		prURL, _ := pr["html_url"].(string)
		ma, _ := pr["merged_at"].(string)
		return &Submission{
			PRNumber:        prNumber,
			PRURL:           prURL,
			SubmittedAt:     ma,
			SubmitterLogin:  login,
			SubmitterName:   displayName,
			SubmitterAvatar: avatar,
			SubmitterProf:   profile,
		}
	}

	// Direct push path.
	login := client.CommitAuthorLogin(targetSHA)
	if login == "" {
		login = client.Owner
	}
	lowered := strings.ToLower(login)
	if _, isMaint := maintainers[lowered]; !isMaint {
		if lowered == "github-actions[bot]" || lowered == "web-flow" || lowered == "dependabot[bot]" {
			return nil
		}
	}

	displayName := login
	avatar := ""
	profile := ""
	if login != "" {
		resolved := cache.Resolve(client, login)
		displayName = resolved["name"]
		avatar = resolved["avatar"]
		profile = resolved["profile"]
	} else {
		displayName = meta.AuthorName
	}

	return &Submission{
		Direct:          true,
		SubmittedAt:     meta.AuthorDate,
		SubmitterLogin:  login,
		SubmitterName:   displayName,
		SubmitterAvatar: avatar,
		SubmitterProf:   profile,
	}
}

// ResolveContributors collects secondary contributors for a module folder.
func ResolveContributors(
	repoRoot, moduleRelPath, submitterLogin string,
	client *GitHubClient,
	cache *UserCache,
) []Contributor {
	if client == nil {
		return nil
	}
	shas := AllCommitsFor(repoRoot, moduleRelPath)
	seen := map[string]struct{}{}
	if submitterLogin != "" {
		seen[strings.ToLower(submitterLogin)] = struct{}{}
	}
	seen["github-actions[bot]"] = struct{}{}
	seen["web-flow"] = struct{}{}

	var out []Contributor
	for _, sha := range shas {
		login := client.CommitAuthorLogin(sha)
		if login == "" {
			continue
		}
		lowered := strings.ToLower(login)
		if _, dup := seen[lowered]; dup {
			continue
		}
		seen[lowered] = struct{}{}
		resolved := cache.Resolve(client, login)
		out = append(out, Contributor{
			Login:   login,
			Name:    resolved["name"],
			Avatar:  resolved["avatar"],
			Profile: resolved["profile"],
		})
	}
	return out
}

// GenerateSubmissions writes modules/submissions.json.
// Returns exit code (0 success, 1 fatal).
func GenerateSubmissions(
	repoRoot, owner, repo, token string,
	maintainers map[string]struct{},
	offline, checkOnly bool,
) int {
	modulesDir := filepath.Join(repoRoot, "modules")
	info, err := os.Stat(modulesDir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "❌ %s does not exist\n", modulesDir)
		return 1
	}
	submissionsPath := filepath.Join(modulesDir, "submissions.json")
	registryPath := filepath.Join(modulesDir, "registry.json")

	var client *GitHubClient
	if !offline {
		client = NewGitHubClient(owner, repo, token)
	}
	cache := NewUserCache()

	registry := ReadRegistry(registryPath)
	pathToID := map[string]string{}
	for _, e := range registry {
		p, _ := e["path"].(string)
		i, _ := e["id"].(string)
		if p != "" && i != "" {
			pathToID[p] = i
		}
	}

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ readdir: %v\n", err)
		return 1
	}
	var folders []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		folders = append(folders, e.Name())
	}
	sort.Strings(folders)

	subs := map[string]any{}
	var skipped []string
	for _, name := range folders {
		id, ok := pathToID[name]
		if !ok {
			skipped = append(skipped, name)
			continue
		}
		rel := "modules/" + name
		s := ResolveForPath(repoRoot, rel, client, maintainers, cache)
		if s == nil {
			skipped = append(skipped, name)
			continue
		}
		s.Contributors = ResolveContributors(repoRoot, rel, s.SubmitterLogin, client, cache)
		subs[id] = s.ToJSON()
	}

	// generatedAt handling.
	existingText, _ := os.ReadFile(submissionsPath)
	existing := map[string]any{}
	if len(existingText) > 0 {
		_ = jsonUnmarshal(existingText, &existing)
	}
	existingSubs, _ := existing["submissions"].(map[string]any)

	var generatedAt string
	if !checkOnly && sameMap(existingSubs, subs) {
		if g, ok := existing["generatedAt"].(string); ok && g != "" {
			generatedAt = g
		}
	}
	if generatedAt == "" {
		generatedAt = nowUTC()
	}

	if checkOnly {
		if sameMap(existingSubs, subs) {
			fmt.Printf("✅ submissions.json is up to date (%d entries)\n", len(subs))
			if len(skipped) > 0 {
				fmt.Printf("   (%d folder(s) hidden from catalog: %s)\n", len(skipped), strings.Join(skipped, ", "))
			}
			return 0
		}
		fmt.Fprintln(os.Stderr, "❌ submissions.json is stale — re-run generate_submissions.py and commit the result.")
		return 1
	}

	out := orderedOutput(generatedAt, subs)
	data, err := jsonMarshalIndent(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ marshal: %v\n", err)
		return 1
	}
	if err := os.WriteFile(submissionsPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ write: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Wrote modules/submissions.json with %d entries\n", len(subs))
	if len(skipped) > 0 {
		fmt.Printf("   (%d folder(s) hidden from catalog: %s)\n", len(skipped), strings.Join(skipped, ", "))
	}
	return 0
}
