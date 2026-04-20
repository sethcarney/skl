package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SyncOptions struct {
	Yes bool
}

func parseSyncOptions(args []string) SyncOptions {
	var opts SyncOptions
	for _, a := range args {
		if a == "--yes" || a == "-y" {
			opts.Yes = true
		}
	}
	return opts
}

func runSync(args []string, opts SyncOptions) {
	cwd, _ := os.Getwd()

	fmt.Printf("\n%sScanning node_modules for skills...%s\n\n", ansiDim, ansiReset)

	found := discoverNodeModuleSkills(cwd)
	if len(found) == 0 {
		fmt.Printf("%sNo skills found in node_modules.%s\n\n", ansiDim, ansiReset)
		fmt.Printf("Install skills packages with npm/yarn/pnpm first, then run %sskills experimental_sync%s\n\n", ansiText, ansiReset)
		return
	}

	// Flatten to *Skill slice, deduplicating by name
	seen := map[string]bool{}
	var skills []*Skill
	for _, f := range found {
		if !seen[f.Skill.Name] {
			seen[f.Skill.Name] = true
			skills = append(skills, f.Skill)
		}
	}

	fmt.Printf("%sFound %d skill(s) in node_modules:%s\n\n", ansiText, len(skills), ansiReset)
	for _, s := range skills {
		shortPath := shortenPath(s.Path, cwd)
		fmt.Printf("  %s%s%s  %s%s%s\n    %s%s%s\n\n",
			ansiText, s.Name, ansiReset,
			ansiDim, s.Description, ansiReset,
			ansiDim, shortPath, ansiReset)
	}

	// Skill selection
	var selectedSkills []*Skill
	if opts.Yes || len(skills) == 1 {
		selectedSkills = skills
	} else {
		options := make([]UIOption, len(skills))
		for i, s := range skills {
			options[i] = UIOption{Label: s.Name, Value: sanitizeName(s.Name), Hint: s.Description}
		}
		var initSel []int
		for i := range skills {
			initSel = append(initSel, i)
		}
		indices, ok := uiMultiselect("Which skills would you like to sync?", options, true, initSel, nil)
		if !ok {
			fmt.Println("Cancelled.")
			return
		}
		for _, i := range indices {
			selectedSkills = append(selectedSkills, skills[i])
		}
	}

	global, mode, agents, ok := promptScopeAndAgents(AddOptions{Yes: opts.Yes}, cwd)
	if !ok {
		return
	}

	fmt.Println()
	for _, skill := range selectedSkills {
		sName := sanitizeName(skill.Name)
		fmt.Printf("%sSyncing %s%s%s...\n", ansiDim, ansiText, skill.Name, ansiReset)

		var failedAgents []string
		for _, agentName := range agents {
			result := installSkillForAgent(skill, agentName, global, mode)
			if !result.Success {
				failedAgents = append(failedAgents, agentName)
			}
		}

		if len(failedAgents) == 0 {
			logSuccess(skill.Name)
		} else {
			logWarn(fmt.Sprintf("%s (failed for: %s)", skill.Name, strings.Join(failedAgents, ", ")))
		}

		// Use relative path from node_modules as source
		relPath, err := filepath.Rel(cwd, skill.Path)
		if err != nil {
			relPath = skill.Path
		}
		relPath = filepath.ToSlash(relPath)

		_ = addSkillToLock(sName, SkillLockEntry{
			Source:     relPath,
			SourceType: string(SourceTypeLocal),
			SourceURL:  relPath,
			PluginName: sName,
		})
		if !global {
			_ = addSkillToLocalLock(sName, LocalSkillLockEntry{
				Source:     relPath,
				SourceType: string(SourceTypeLocal),
			}, cwd)
		}
	}

	fmt.Println()
	printInstallSummary(len(selectedSkills), global, agents, mode)
}
