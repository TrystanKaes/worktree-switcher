package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const copyConfigFilename = ".worktree-switcher"

func warnCopy(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "wt: copy: "+format+"\n", args...)
}

func loadCopyConfig(sourceWorktreePath string) ([]string, error) {
	configPath := filepath.Join(sourceWorktreePath, copyConfigFilename)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var paths []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		paths = append(paths, filepath.Clean(line))
	}

	return paths, nil
}

func findMainWorktreePath(worktrees []Worktree, mainBranch string) string {
	for _, wt := range worktrees {
		if wt.IsBare {
			continue
		}
		if wt.Branch == mainBranch {
			return wt.Path
		}
	}
	return ""
}

func copyFilesFromMain(destPath string) {
	mainBranch := detectMainBranch()
	if mainBranch == "" {
		return
	}

	worktrees, err := ListWorktrees()
	if err != nil {
		warnCopy("failed to list worktrees: %v", err)
		return
	}

	mainPath := findMainWorktreePath(worktrees, mainBranch)
	if mainPath == "" {
		warnCopy("could not find main worktree for branch %q", mainBranch)
		return
	}

	copyFilesFromSource(mainPath, destPath, "main worktree")
}

func copyFilesFromSource(sourcePath, destPath, sourceLabel string) {
	if filepath.Clean(sourcePath) == filepath.Clean(destPath) {
		return
	}

	paths, err := loadCopyConfig(sourcePath)
	if err != nil {
		warnCopy("failed to load %s: %v", copyConfigFilename, err)
		return
	}
	if len(paths) == 0 {
		return
	}

	copied := 0
	for _, p := range paths {
		if !isValidCopyPath(p) {
			warnCopy("invalid path %q in %s (must be relative and not contain '..')", p, copyConfigFilename)
			continue
		}

		src := filepath.Join(sourcePath, p)
		dst := filepath.Join(destPath, p)

		info, err := os.Lstat(src)
		if err != nil {
			warnCopy("cannot stat %q: %v", src, err)
			continue
		}

		if err := copyPath(src, dst); err != nil {
			warnCopy("failed to copy %q -> %q: %v", src, dst, err)
			continue
		}

		if info.IsDir() {
			copied += countCopiedFiles(src)
		} else {
			copied++
		}
	}

	if copied > 0 {
		fmt.Fprintf(os.Stderr, "copied %d files from %s\n", copied, sourceLabel)
	}
}

func isValidCopyPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || filepath.IsAbs(p) {
		return false
	}
	if strings.Contains(p, "..") {
		return false
	}
	return true
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	mode := info.Mode()
	switch {
	case mode&os.ModeSymlink != 0:
		return copySymlink(src, dst)
	case info.IsDir():
		return copyDir(src, dst, mode.Perm())
	default:
		return copyFile(src, dst, mode.Perm())
	}
}

func copyDir(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(dst, perm); err != nil {
		return err
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		info, err := os.Lstat(path)
		if err != nil {
			return err
		}

		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0:
			return copySymlink(path, target)
		case d.IsDir():
			return os.MkdirAll(target, mode.Perm())
		default:
			return copyFile(path, target, mode.Perm())
		}
	})
}

func copyFile(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return os.Chmod(dst, perm)
}

func copySymlink(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	target, err := os.Readlink(src)
	if err != nil {
		return err
	}

	_ = os.RemoveAll(dst)
	return os.Symlink(target, dst)
}

func countCopiedFiles(path string) int {
	count := 0
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	return count
}
