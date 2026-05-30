package runtimeskills

import (
	"bytes"
	"fmt"
	"pa/internal/prompt"
	"pa/internal/toolcatalog"
	"strings"

	"gopkg.in/yaml.v3"
)

// Package is one runtime skill loaded from a subdirectory (EP-013).
type Package struct {
	ID          string
	Name        string
	Description string
	Tools       []string
	Body        string
}

// LoadDir loads immediate subdirectories of root that contain SKILL.md.
func LoadDir(root string) ([]*Package, error) {
	entries, err := listSkillDirs(root)
	if err != nil {
		return nil, err
	}
	var out []*Package
	for _, dir := range entries {
		p, err := loadPackageDir(dir)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

type frontMatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools"`
}

func loadPackageDir(dir string) (*Package, error) {
	path := join(dir, "SKILL.md")
	raw, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("skill %s: %w", dir, err)
	}
	if prompt.TextContainsForbiddenMarkerLine(string(raw)) {
		return nil, fmt.Errorf("skill %s: SKILL.md contains a forbidden PA marker line", dir)
	}
	fm, body, err := parseFrontmatter(raw)
	if err != nil {
		return nil, fmt.Errorf("skill %s: %w", dir, err)
	}
	name := strings.TrimSpace(fm.Name)
	desc := strings.TrimSpace(fm.Description)
	if name == "" || desc == "" {
		return nil, fmt.Errorf("skill %s: SKILL.md frontmatter requires name and description", dir)
	}
	id := baseName(dir)
	var tools []string
	for _, t := range fm.Tools {
		t = strings.TrimSpace(t)
		if t != "" {
			tools = append(tools, t)
		}
	}
	return &Package{
		ID:          id,
		Name:        name,
		Description: desc,
		Tools:       tools,
		Body:        strings.TrimSpace(body),
	}, nil
}

func parseFrontmatter(raw []byte) (frontMatter, string, error) {
	var fm frontMatter
	if !bytes.HasPrefix(raw, []byte("---\n")) {
		return fm, "", fmt.Errorf("SKILL.md must start with YAML frontmatter ---")
	}
	rest := raw[4:]
	end := bytes.Index(rest, []byte("\n---\n"))
	if end < 0 {
		return fm, "", fmt.Errorf("SKILL.md: missing closing --- frontmatter delimiter")
	}
	if err := yaml.Unmarshal(rest[:end], &fm); err != nil {
		return fm, "", fmt.Errorf("frontmatter yaml: %w", err)
	}
	body := string(rest[end+5:])
	return fm, body, nil
}

// EmbeddingText is the string embedded for vec_skills (name, description, body).
func (p *Package) EmbeddingText() string {
	var b strings.Builder
	b.WriteString(p.Name)
	b.WriteByte('\n')
	b.WriteString(p.Description)
	b.WriteByte('\n')
	b.WriteString(p.Body)
	return b.String()
}

// PlaybookText is full markdown injected into the SKILLS marker block (title + body).
func (p *Package) PlaybookText() string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(p.Name)
	b.WriteString("\n\n")
	b.WriteString(p.Body)
	return b.String()
}

// ValidateToolRefs returns an error if any declared tool id is missing from catalog or native allowlist.
func ValidateToolRefs(pkgs []*Package, cat *toolcatalog.Catalog, nativeIDs []string) error {
	nativeSet := make(map[string]struct{}, len(nativeIDs))
	for _, id := range nativeIDs {
		nativeSet[id] = struct{}{}
	}
	for _, p := range pkgs {
		for _, tid := range p.Tools {
			if _, inCat := cat.Tools[tid]; inCat {
				continue
			}
			if _, ok := nativeSet[tid]; ok {
				continue
			}
			return fmt.Errorf("skill %q: unknown tool id %q (not in catalog or allowed native tools)", p.ID, tid)
		}
	}
	return nil
}

// join, listSkillDirs, readFile, baseName - use os/path/filepath in load_unix.go or inline
