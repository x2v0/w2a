package modules

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// GitHubClient is a minimal REST client using net/http (stdlib only).
type GitHubClient struct {
	Owner   string
	Repo    string
	Token   string
	Timeout time.Duration
	http    *http.Client
}

func NewGitHubClient(owner, repo, token string) *GitHubClient {
	return &GitHubClient{
		Owner:   owner,
		Repo:    repo,
		Token:   token,
		Timeout: 15 * time.Second,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *GitHubClient) request(rawURL string, out any) error {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "w2a-submissions-generator")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var errNotFound = fmt.Errorf("not found")

// PullsForCommit returns PRs associated with a commit SHA (may be empty).
func (c *GitHubClient) PullsForCommit(sha string) []map[string]any {
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s/pulls", c.Owner, c.Repo, sha)
	var out []map[string]any
	if err := c.request(u, &out); err != nil {
		if err == errNotFound {
			return nil
		}
		fmt.Fprintf(os.Stderr, "⚠️  GitHub API error for commit %s: %v\n", sha[:min(7, len(sha))], err)
		return nil
	}
	return out
}

// User resolves a GitHub login to its full user payload (may be nil).
func (c *GitHubClient) User(login string) map[string]any {
	u := "https://api.github.com/users/" + url.PathEscape(login)
	var out map[string]any
	if err := c.request(u, &out); err != nil {
		return nil
	}
	return out
}

// CommitAuthorLogin resolves a commit author's GitHub login.
func (c *GitHubClient) CommitAuthorLogin(sha string) string {
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", c.Owner, c.Repo, sha)
	var out map[string]any
	if err := c.request(u, &out); err != nil {
		return ""
	}
	author, _ := out["author"].(map[string]any)
	login, _ := author["login"].(string)
	return login
}

// GetPull fetches a single PR by number (may be nil).
func (c *GitHubClient) GetPull(number int) map[string]any {
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", c.Owner, c.Repo, number)
	var out map[string]any
	if err := c.request(u, &out); err != nil {
		if err != errNotFound {
			fmt.Fprintf(os.Stderr, "⚠️  GitHub API error for PR #%d: %v\n", number, err)
		}
		return nil
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
