package core

import (
	"pa/internal/config"
	"pa/internal/runtimeskills"
	"pa/internal/toolcatalog"
	"strings"
	"unicode/utf8"
)

type toolOrigin uint8

const (
	originVector toolOrigin = 1 << 0
	originAlways toolOrigin = 1 << 1
	originSkill  toolOrigin = 1 << 2
)

func mergeToolIDs(rs *config.RuntimeSkillsConfig, skills []*runtimeskills.Package, vectorIDs []string) (ordered []string, sources map[string]toolOrigin) {
	sources = make(map[string]toolOrigin)
	add := func(id string, o toolOrigin) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := sources[id]; !ok {
			ordered = append(ordered, id)
		}
		sources[id] |= o
	}
	if rs != nil && rs.Enabled {
		for _, id := range rs.AlwaysInclude {
			add(id, originAlways)
		}
		for _, p := range skills {
			for _, tid := range p.Tools {
				add(tid, originSkill)
			}
		}
	}
	for _, id := range vectorIDs {
		add(id, originVector)
	}
	return ordered, sources
}

func trimSkillPackagesByBudget(pkgs []*runtimeskills.Package, maxRunes int) []*runtimeskills.Package {
	if maxRunes < 1 || len(pkgs) == 0 {
		return pkgs
	}
	var kept []*runtimeskills.Package
	total := 0
	for _, p := range pkgs {
		n := utf8.RuneCountInString(p.PlaybookText())
		if total+n > maxRunes {
			if len(kept) == 0 {
				// First (highest-ranked) skill alone exceeds budget: omit all (REQ-13.012).
				return nil
			}
			break
		}
		kept = append(kept, p)
		total += n
	}
	return kept
}

func trimToolIDsForInstructionBudget(cat *toolcatalog.Catalog, ids []string, sources map[string]toolOrigin, maxRunes int) []string {
	if cat == nil || len(ids) == 0 || maxRunes < 1 {
		return ids
	}
	cur := append([]string(nil), ids...)
	for utf8.RuneCountInString(toolcatalog.AggregateSystemPrompts(cat, cur)) > maxRunes {
		removed := false
		for i := len(cur) - 1; i >= 0; i-- {
			id := cur[i]
			if sources[id] == originVector {
				cur = append(cur[:i], cur[i+1:]...)
				delete(sources, id)
				removed = true
				break
			}
		}
		if !removed {
			break
		}
	}
	return cur
}
