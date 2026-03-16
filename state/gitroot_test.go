package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitRootWithRepo(t *testing.T) {
	// Create a temp dir with a .git directory to simulate a repo.
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// A subdirectory should resolve to root.
	sub := filepath.Join(root, "src", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(sub)
	if got != root {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, root)
	}
}

func TestFindGitRootAtRoot(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(root)
	if got != root {
		t.Errorf("findGitRoot(%q) = %q, want %q", root, got, root)
	}
}

func TestFindGitRootNoRepo(t *testing.T) {
	// Temp dir with no .git — should return the dir itself.
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(sub)
	// Should be the absolute path of sub since no .git found.
	abs, _ := filepath.Abs(sub)
	if got != abs {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, abs)
	}
}

func TestFindGitRootWorktree(t *testing.T) {
	// .git can be a file (worktree or submodule).
	root := t.TempDir()
	gitFile := filepath.Join(root, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /some/other/path"), 0o644); err != nil {
		t.Fatal(err)
	}

	sub := filepath.Join(root, "deep", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findGitRoot(sub)
	if got != root {
		t.Errorf("findGitRoot(%q) = %q, want %q", sub, got, root)
	}
}

func TestGitRootCache(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sub := filepath.Join(root, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	cache := newGitRootCache()

	// First call should resolve.
	got1 := cache.Resolve(sub)
	if got1 != root {
		t.Errorf("first Resolve() = %q, want %q", got1, root)
	}

	// Second call should use cache (same result).
	got2 := cache.Resolve(sub)
	if got2 != root {
		t.Errorf("cached Resolve() = %q, want %q", got2, root)
	}
}

func TestGitRootCacheEmptyDir(t *testing.T) {
	cache := newGitRootCache()
	got := cache.Resolve("")
	if got != "" {
		t.Errorf("Resolve(\"\") = %q, want empty", got)
	}
}

func TestProjectName(t *testing.T) {
	tests := []struct {
		gitRoot string
		want    string
	}{
		{"/home/user/projects/my-app", "my-app"},
		{"/opt/code/backend", "backend"},
		{"", "unknown"},
		{"/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.gitRoot, func(t *testing.T) {
			got := projectName(tt.gitRoot)
			if got != tt.want {
				t.Errorf("projectName(%q) = %q, want %q", tt.gitRoot, got, tt.want)
			}
		})
	}
}
