package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func runListWithOpts(globalFlag *bool, agentFilter []string, jsonMode bool) {
	skills, err := listInstalledSkills(globalFlag, agentFilter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(skills) == 0 {
		if jsonMode {
			fmt.Println("[]")
		} else {
			fmt.Printf("%sNo skills installed.%s\n\n", ansiDim, ansiReset)
			fmt.Printf("Add your first skill with %sskl add <package>%s\n", ansiText, ansiReset)
		}
		return
	}

	// Sort by scope then name
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Scope != skills[j].Scope {
			return skills[i].Scope < skills[j].Scope
		}
		return skills[i].Name < skills[j].Name
	})

	if jsonMode {
		out, _ := json.MarshalIndent(skills, "", "  ")
		fmt.Println(string(out))
		return
	}

	cwd, _ := os.Getwd()

	// Group by scope
	byScope := map[string][]*InstalledSkill{}
	for _, s := range skills {
		byScope[s.Scope] = append(byScope[s.Scope], s)
	}

	scopes := []string{"project", "global"}
	for _, scope := range scopes {
		scopeSkills, ok := byScope[scope]
		if !ok {
			continue
		}
		scopeTitle := strings.ToUpper(scope[:1]) + scope[1:]
		fmt.Printf("%s%s skills:%s\n\n", ansiText, scopeTitle, ansiReset)
		for _, s := range scopeSkills {
			fmt.Printf("  %s%s%s", ansiText, s.Name, ansiReset)
			if s.Description != "" {
				fmt.Printf("  %s%s%s", ansiDim, s.Description, ansiReset)
			}
			fmt.Println()
			if len(s.Agents) > 0 {
				var displayNames []string
				for _, a := range s.Agents {
					if cfg := allAgents[a]; cfg != nil {
						displayNames = append(displayNames, cfg.DisplayName)
					} else {
						displayNames = append(displayNames, a)
					}
				}
				fmt.Printf("    %sagents: %s%s\n", ansiDim, strings.Join(displayNames, ", "), ansiReset)
			}
			shortPath := shortenPath(s.Path, cwd)
			fmt.Printf("    %s%s%s\n", ansiDim, shortPath, ansiReset)
		}
		fmt.Println()
	}
}
