package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RemoveOptions struct {
	Global  bool
	Agents  []string
	Skills  []string
	Yes     bool
	All     bool
}

func parseRemoveOptions(args []string) ([]string, RemoveOptions) {
	var positional []string
	var opts RemoveOptions
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--global" || a == "-g":
			opts.Global = true
		case a == "--yes" || a == "-y":
			opts.Yes = true
		case a == "--all":
			opts.All = true
		case a == "--agent" || a == "-a":
			i++
			for i < len(args) && !strings.HasPrefix(args[i], "-") {
				opts.Agents = append(opts.Agents, args[i])
				i++
			}
			i--
		case a == "--skill" || a == "-s":
			i++
			for i < len(args) && !strings.HasPrefix(args[i], "-") {
				opts.Skills = append(opts.Skills, args[i])
				i++
			}
			i--
		default:
			if !strings.HasPrefix(a, "-") {
				positional = append(positional, a)
			}
		}
	}
	if opts.All {
		opts.Skills = []string{"*"}
		opts.Agents = []string{"*"}
		opts.Yes = true
	}
	return positional, opts
}

func runRemove(positional []string, opts RemoveOptions) {
	cwd, _ := os.Getwd()
	global := opts.Global

	// Merge positional + --skill args
	skillFilter := opts.Skills
	if len(positional) > 0 {
		skillFilter = append(skillFilter, positional...)
	}

	// Determine scope if not specified
	if !opts.Global && !opts.Yes {
		idx, ok := uiSelect("Which scope?", []UIOption{
			{Label: "Project", Hint: "remove from this project"},
			{Label: "Global", Hint: "remove from your user account"},
		})
		if !ok {
			return
		}
		global = idx == 1
	}

	// Discover installed skills in the appropriate scope
	scopeGlobal := &global
	installed, err := listInstalledSkills(scopeGlobal, opts.Agents)
	if err != nil || len(installed) == 0 {
		fmt.Printf("%sNo skills installed.%s\n", ansiDim, ansiReset)
		return
	}

	// If skill filter provided, narrow down
	var toRemove []*InstalledSkill
	if len(skillFilter) > 0 && !(len(skillFilter) == 1 && skillFilter[0] == "*") {
		for _, s := range installed {
			for _, f := range skillFilter {
				if strings.EqualFold(s.Name, f) || strings.EqualFold(sanitizeName(s.Name), sanitizeName(f)) {
					toRemove = append(toRemove, s)
					break
				}
			}
		}
		if len(toRemove) == 0 {
			fmt.Printf("%sNo matching skills found.%s\n", ansiDim, ansiReset)
			return
		}
	} else if len(skillFilter) == 1 && skillFilter[0] == "*" {
		toRemove = installed
	} else if opts.Yes || len(installed) == 1 {
		toRemove = installed
	} else {
		// Interactive selection
		options := make([]UIOption, len(installed))
		for i, s := range installed {
			hint := s.Description
			if len(s.Agents) > 0 {
				hint = strings.Join(s.Agents, ", ")
			}
			options[i] = UIOption{Label: s.Name, Value: sanitizeName(s.Name), Hint: hint}
		}
		indices, ok := uiMultiselect("Which skills would you like to remove?", options, true, nil, nil)
		if !ok {
			fmt.Println("Cancelled.")
			return
		}
		for _, i := range indices {
			toRemove = append(toRemove, installed[i])
		}
	}

	if len(toRemove) == 0 {
		return
	}

	// Confirm
	if !opts.Yes && len(toRemove) > 0 {
		var names []string
		for _, s := range toRemove {
			names = append(names, s.Name)
		}
		confirmed, ok := uiConfirm(fmt.Sprintf("Remove %d skill(s): %s?", len(toRemove), strings.Join(names, ", ")))
		if !ok || !confirmed {
			fmt.Println("Cancelled.")
			return
		}
	}

	fmt.Println()

	// Remove each skill
	for _, skill := range toRemove {
		sName := sanitizeName(skill.Name)

		// Remove from each agent's skills directory
		agentsToRemove := skill.Agents
		if len(opts.Agents) > 0 {
			agentsToRemove = opts.Agents
		}

		for _, agentName := range agentsToRemove {
			agentBase := getAgentBaseDir(agentName, global, cwd)
			if agentBase == "" {
				continue
			}
			// Try both the sanitized name and the directory name
			for _, name := range []string{sName, filepath.Base(skill.Path)} {
				agentSkillDir := filepath.Join(agentBase, name)
				if !isPathSafe(agentBase, agentSkillDir) {
					continue
				}
				if info, err := os.Lstat(agentSkillDir); err == nil {
					if info.Mode()&os.ModeSymlink != 0 {
						os.Remove(agentSkillDir)
					} else {
						os.RemoveAll(agentSkillDir)
					}
				}
			}
		}

		// Remove canonical directory
		canonicalDir := getCanonicalPath(skill.Name, global)
		if canonicalDir != "" && isPathSafe(getCanonicalSkillsDir(global, cwd), canonicalDir) {
			os.RemoveAll(canonicalDir)
		}

		// Update lock files
		_ = removeSkillFromLock(sName)
		if !global {
			_ = removeSkillFromLocalLock(sName, cwd)
		}

		logSuccess("Removed " + skill.Name)
	}

	fmt.Println()
}
