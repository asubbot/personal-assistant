package runtimeskills

import (
	"os"
	"path/filepath"
	"sort"
)

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func join(elem ...string) string {
	return filepath.Join(elem...)
}

func baseName(path string) string {
	return filepath.Base(path)
}

func listSkillDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		dir := filepath.Join(root, name)
		st, err := os.Stat(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if st.IsDir() {
			continue
		}
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs, nil
}

// DirExists reports whether path is a directory (for config validation).
func DirExists(path string) (bool, error) {
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// error-masked-as-false-bool: safe — a missing path is the expected negative predicate result
			return false, nil
		}
		return false, err
	}
	return st.IsDir(), nil
}
