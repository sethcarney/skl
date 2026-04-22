package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Skill struct {
	Name        string
	Description string
	Path        string
	RawContent  string
	PluginName  string
	Metadata    map[string]interface{}
}

func parseFrontmatter(raw string) (data map[string]interface{}, content string) {
	// Match ---\n...\n---\n
	const delim = "---"
	if !strings.HasPrefix(raw, delim) {
		return map[string]interface{}{}, raw
	}
	rest := raw[len(delim):]
	// skip \r\n or \n
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else {
		return map[string]interface{}{}, raw
	}

	// find closing ---
	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return map[string]interface{}{}, raw
	}
	yamlPart := rest[:endIdx]
	afterDelim := rest[endIdx+4:] // skip \n---
	if strings.HasPrefix(afterDelim, "\r\n") {
		afterDelim = afterDelim[2:]
	} else if strings.HasPrefix(afterDelim, "\n") {
		afterDelim = afterDelim[1:]
	}

	var d map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlPart), &d); err != nil {
		return map[string]interface{}{}, raw
	}
	if d == nil {
		d = map[string]interface{}{}
	}
	return d, afterDelim
}

func parseSkillMd(skillMdPath string, includeInternal bool) (*Skill, error) {
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, err
	}
	raw := string(content)
	data, _ := parseFrontmatter(raw)

	nameVal, ok1 := data["name"]
	descVal, ok2 := data["description"]
	if !ok1 || !ok2 {
		return nil, nil
	}
	name, ok := nameVal.(string)
	if !ok || name == "" {
		return nil, nil
	}
	desc, ok := descVal.(string)
	if !ok {
		return nil, nil
	}

	// Check internal flag
	if meta, ok := data["metadata"]; ok {
		if metaMap, ok := meta.(map[string]interface{}); ok {
			if internal, ok := metaMap["internal"]; ok {
				if b, ok := internal.(bool); ok && b {
					if !includeInternal && os.Getenv("INSTALL_INTERNAL_SKILLS") != "1" && os.Getenv("INSTALL_INTERNAL_SKILLS") != "true" {
						return nil, nil
					}
				}
			}
		}
	}

	var metaMap map[string]interface{}
	if m, ok := data["metadata"]; ok {
		if mm, ok := m.(map[string]interface{}); ok {
			metaMap = mm
		}
	}

	return &Skill{
		Name:        name,
		Description: desc,
		Path:        filepath.Dir(skillMdPath),
		RawContent:  raw,
		Metadata:    metaMap,
	}, nil
}

var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
}

func hasSkillMd(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !info.IsDir()
}

func findSkillDirs(dir string, depth, maxDepth int) []string {
	if depth > maxDepth {
		return nil
	}
	var result []string
	if hasSkillMd(dir) {
		result = append(result, dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}
	for _, e := range entries {
		if !e.IsDir() || skipDirs[e.Name()] {
			continue
		}
		sub := findSkillDirs(filepath.Join(dir, e.Name()), depth+1, maxDepth)
		result = append(result, sub...)
	}
	return result
}

func isSubpathSafe(basePath, subpath string) bool {
	base, _ := filepath.Abs(basePath)
	target, _ := filepath.Abs(filepath.Join(basePath, subpath))
	return target == base || strings.HasPrefix(target, base+string(filepath.Separator))
}

type DiscoverOptions struct {
	IncludeInternal bool
	FullDepth       bool
}

// getPluginGroupings returns a map of skill dir path -> plugin name, based on
// plugin-manifest files (.claude-plugin/marketplace.json) in the search path.
// This is a simplified implementation of the TypeScript getPluginGroupings.
func getPluginGroupings(searchPath string) map[string]string {
	result := map[string]string{}
	manifestPath := filepath.Join(searchPath, ".claude-plugin", "marketplace.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return result
	}
	var manifest struct {
		Name   string `yaml:"name"`
		Plugins []struct {
			Name     string `yaml:"name"`
			SkillDir string `yaml:"skillDir"`
		} `yaml:"plugins"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return result
	}
	pluginName := manifest.Name
	if pluginName == "" {
		return result
	}
	// If plugins list provided, each plugin's skillDir maps to its own name or parent
	if len(manifest.Plugins) > 0 {
		for _, p := range manifest.Plugins {
			if p.SkillDir != "" {
				abs, _ := filepath.Abs(filepath.Join(searchPath, p.SkillDir))
				name := p.Name
				if name == "" {
					name = pluginName
				}
				result[abs] = name
			}
		}
	}
	return result
}

// getPluginSkillPaths returns extra skill search dirs from plugin manifests.
func getPluginSkillPaths(searchPath string) []string {
	var result []string
	manifestPath := filepath.Join(searchPath, ".claude-plugin", "marketplace.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return result
	}
	var manifest struct {
		SkillDirs []string `yaml:"skillDirs"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return result
	}
	for _, d := range manifest.SkillDirs {
		result = append(result, filepath.Join(searchPath, d))
	}
	return result
}

func discoverSkills(basePath, subpath string, opts DiscoverOptions) ([]*Skill, error) {
	var skills []*Skill
	seenNames := map[string]bool{}

	if subpath != "" && !isSubpathSafe(basePath, subpath) {
		return nil, fmt.Errorf("invalid subpath: %q escapes repository directory", subpath)
	}

	searchPath := basePath
	if subpath != "" {
		searchPath = filepath.Join(basePath, subpath)
	}

	pluginGroupings := getPluginGroupings(searchPath)

	enhanceSkill := func(s *Skill) *Skill {
		resolvedPath, _ := filepath.Abs(s.Path)
		if pn, ok := pluginGroupings[resolvedPath]; ok {
			s.PluginName = pn
		}
		return s
	}

	// If pointing directly at a skill
	if hasSkillMd(searchPath) {
		skill, err := parseSkillMd(filepath.Join(searchPath, "SKILL.md"), opts.IncludeInternal)
		if err == nil && skill != nil {
			skill = enhanceSkill(skill)
			skills = append(skills, skill)
			seenNames[skill.Name] = true
			if !opts.FullDepth {
				return skills, nil
			}
		}
	}

	// Priority search directories
	priorityDirs := []string{
		searchPath,
		filepath.Join(searchPath, "skills"),
		filepath.Join(searchPath, "skills/.curated"),
		filepath.Join(searchPath, "skills/.experimental"),
		filepath.Join(searchPath, "skills/.system"),
		filepath.Join(searchPath, ".agents/skills"),
		filepath.Join(searchPath, ".claude/skills"),
		filepath.Join(searchPath, ".cline/skills"),
		filepath.Join(searchPath, ".codebuddy/skills"),
		filepath.Join(searchPath, ".codex/skills"),
		filepath.Join(searchPath, ".commandcode/skills"),
		filepath.Join(searchPath, ".continue/skills"),
		filepath.Join(searchPath, ".github/skills"),
		filepath.Join(searchPath, ".goose/skills"),
		filepath.Join(searchPath, ".iflow/skills"),
		filepath.Join(searchPath, ".junie/skills"),
		filepath.Join(searchPath, ".kilocode/skills"),
		filepath.Join(searchPath, ".kiro/skills"),
		filepath.Join(searchPath, ".mux/skills"),
		filepath.Join(searchPath, ".neovate/skills"),
		filepath.Join(searchPath, ".opencode/skills"),
		filepath.Join(searchPath, ".openhands/skills"),
		filepath.Join(searchPath, ".pi/skills"),
		filepath.Join(searchPath, ".qoder/skills"),
		filepath.Join(searchPath, ".roo/skills"),
		filepath.Join(searchPath, ".trae/skills"),
		filepath.Join(searchPath, ".windsurf/skills"),
		filepath.Join(searchPath, ".zencoder/skills"),
	}

	// Add plugin manifest skill dirs
	priorityDirs = append(priorityDirs, getPluginSkillPaths(searchPath)...)

	for _, dir := range priorityDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillDir := filepath.Join(dir, e.Name())
			if hasSkillMd(skillDir) {
				skill, err := parseSkillMd(filepath.Join(skillDir, "SKILL.md"), opts.IncludeInternal)
				if err == nil && skill != nil && !seenNames[skill.Name] {
					skill = enhanceSkill(skill)
					skills = append(skills, skill)
					seenNames[skill.Name] = true
				}
			}
		}
	}

	// Fall back to recursive search if nothing found or fullDepth
	if len(skills) == 0 || opts.FullDepth {
		allSkillDirs := findSkillDirs(searchPath, 0, 5)
		for _, skillDir := range allSkillDirs {
			skill, err := parseSkillMd(filepath.Join(skillDir, "SKILL.md"), opts.IncludeInternal)
			if err == nil && skill != nil && !seenNames[skill.Name] {
				skill = enhanceSkill(skill)
				skills = append(skills, skill)
				seenNames[skill.Name] = true
			}
		}
	}

	return skills, nil
}

func getSkillDisplayName(skill *Skill) string {
	if skill.Name != "" {
		return skill.Name
	}
	return filepath.Base(skill.Path)
}

func filterSkills(skills []*Skill, inputNames []string) []*Skill {
	var result []*Skill
	normalized := make([]string, len(inputNames))
	for i, n := range inputNames {
		normalized[i] = strings.ToLower(n)
	}
	for _, s := range skills {
		name := strings.ToLower(s.Name)
		display := strings.ToLower(getSkillDisplayName(s))
		for _, input := range normalized {
			if input == name || input == display {
				result = append(result, s)
				break
			}
		}
	}
	return result
}

type NodeModuleSkill struct {
	Skill       *Skill
	PackageName string
}

// discoverNodeModuleSkills scans node_modules for SKILL.md files.
func discoverNodeModuleSkills(cwd string) []NodeModuleSkill {
	var results []NodeModuleSkill

	nmDir := filepath.Join(cwd, "node_modules")
	entries, err := os.ReadDir(nmDir)
	if err != nil {
		return nil
	}

	processPackage := func(pkgDir, packageName string) {
		// Check root SKILL.md
		if s, err := parseSkillMd(filepath.Join(pkgDir, "SKILL.md"), false); err == nil && s != nil {
			results = append(results, NodeModuleSkill{s, packageName})
			return
		}
		// Check common locations
		for _, dir := range []string{pkgDir, filepath.Join(pkgDir, "skills"), filepath.Join(pkgDir, ".agents/skills")} {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				skillDir := filepath.Join(dir, e.Name())
				if s, err := parseSkillMd(filepath.Join(skillDir, "SKILL.md"), false); err == nil && s != nil {
					results = append(results, NodeModuleSkill{s, packageName})
				}
			}
		}
	}

	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fullPath := filepath.Join(nmDir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		if e.Name()[0] == '@' && info.IsDir() {
			// Scoped package
			subEntries, err := os.ReadDir(fullPath)
			if err != nil {
				continue
			}
			for _, sub := range subEntries {
				if sub.IsDir() {
					processPackage(filepath.Join(fullPath, sub.Name()), e.Name()+"/"+sub.Name())
				}
			}
		} else if info.IsDir() {
			processPackage(fullPath, e.Name())
		}
	}

	return results
}

// Needed for the discoverNodeModuleSkills return
var _ fs.DirEntry // suppress unused import warning
