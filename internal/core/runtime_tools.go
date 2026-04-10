package core

import (
	"pa/internal/config"
	"pa/internal/runtimeskills"
	"strings"
)

type toolOrigin uint8

const (
	originVector toolOrigin = 1 << 0
	originAlways toolOrigin = 1 << 1
	originSkill  toolOrigin = 1 << 2
)

func mergeToolIDs(tc *config.ToolsConfig, rs *config.RuntimeSkillsConfig, skills []*runtimeskills.Package, vectorIDs []string) (ordered []string, sources map[string]toolOrigin) {
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
	if tc != nil {
		for _, id := range tc.AlwaysInclude {
			add(id, originAlways)
		}
	}
	if rs != nil && rs.Enabled {
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
