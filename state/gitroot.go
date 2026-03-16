package state

import (
	"os"
	"path/filepath"
	"sync"
)

// gitRootCache caches the mapping from directory path to git root so we
// don't stat the filesystem repeatedly for the same directories.
type gitRootCache struct {
	mu    sync.RWMutex
	cache map[string]string
}

// newGitRootCache creates a new empty cache.
func newGitRootCache() *gitRootCache {
	return &gitRootCache{cache: make(map[string]string)}
}

// Resolve finds the git root for the given directory. It walks parent
// directories looking for a .git entry. The result is cached.
// If no git root is found, dir itself is returned.
func (c *gitRootCache) Resolve(dir string) string {
	if dir == "" {
		return ""
	}

	c.mu.RLock()
	if root, ok := c.cache[dir]; ok {
		c.mu.RUnlock()
		return root
	}
	c.mu.RUnlock()

	root := findGitRoot(dir)

	c.mu.Lock()
	c.cache[dir] = root
	c.mu.Unlock()

	return root
}

// findGitRoot walks up from dir looking for a .git directory or file.
// Returns dir itself if no git root is found.
func findGitRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}

	current := abs
	for {
		gitPath := filepath.Join(current, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			// .git can be a directory (normal repo) or a file (worktree/submodule).
			if info.IsDir() || info.Mode().IsRegular() {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root without finding .git.
			return abs
		}
		current = parent
	}
}

// projectName derives a short display name from a git root path.
func projectName(gitRoot string) string {
	if gitRoot == "" {
		return "unknown"
	}
	return filepath.Base(gitRoot)
}
