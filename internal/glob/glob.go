// Package glob provides case-insensitive glob matching for file names and
// slash-separated relative paths. Windows file systems are case-insensitive
// and the tool's users are document people, so "Thumbs.db" must match
// "thumbs.db" everywhere.
package glob

import (
	"path"
	"strings"
)

// Match reports whether name matches the shell pattern (see path.Match),
// ignoring case. Invalid patterns never match.
func Match(pattern, name string) bool {
	ok, err := path.Match(strings.ToLower(pattern), strings.ToLower(name))
	return err == nil && ok
}

// MatchAny reports whether any pattern matches the entry. A pattern that
// contains a slash is matched against the slash-separated relative path
// (and also treated as a directory prefix: "a/b" matches "a/b/c.txt").
// A pattern without a slash matches the base name or any folder on the
// relative path, so "Старое" excludes that folder wherever it sits. A
// trailing slash is allowed and ignored (gitignore habit).
func MatchAny(patterns []string, rel, name string) bool {
	var components []string
	for _, p := range patterns {
		p = strings.TrimSuffix(strings.ReplaceAll(p, "\\", "/"), "/")
		if p == "" {
			continue
		}
		if strings.Contains(p, "/") {
			if Match(p, rel) {
				return true
			}
			lp, lr := strings.ToLower(p), strings.ToLower(rel)
			if lr == lp || strings.HasPrefix(lr, lp+"/") {
				return true
			}
			continue
		}
		if Match(p, name) {
			return true
		}
		if components == nil {
			components = strings.Split(rel, "/")
		}
		for _, c := range components[:max(len(components)-1, 0)] {
			if Match(p, c) {
				return true
			}
		}
	}
	return false
}
