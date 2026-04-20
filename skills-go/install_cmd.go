package main

import (
	"fmt"
	"os"
	"strings"
)

// experimental_install: restore skills from skills-lock.json

func runInstallFromLock(args []string) {
	var yes bool
	for _, a := range args {
		if a == "--yes" || a == "-y" {
			yes = true
		}
	}

	cwd, _ := os.Getwd()
	lock := readLocalLock(cwd)

	if len(lock.Skills) == 0 {
		fmt.Printf("\n%sNo skills in skills-lock.json.%s\n\n", ansiDim, ansiReset)
		fmt.Printf("Add skills with %sskills add <package>%s\n\n", ansiText, ansiReset)
		return
	}

	fmt.Printf("\n%sRestoring %d skill(s) from skills-lock.json...%s\n\n", ansiText, len(lock.Skills), ansiReset)

	// Group by source to batch installations
	type sourceGroup struct {
		source     string
		sourceType string
		ref        string
		skills     []string
	}
	sourceMap := map[string]*sourceGroup{}
	for sName, entry := range lock.Skills {
		key := entry.Source + "|" + entry.Ref
		if g, ok := sourceMap[key]; ok {
			g.skills = append(g.skills, sName)
		} else {
			sourceMap[key] = &sourceGroup{
				source:     entry.Source,
				sourceType: entry.SourceType,
				ref:        entry.Ref,
				skills:     []string{sName},
			}
		}
	}

	for _, group := range sourceMap {
		fmt.Printf("%sInstalling from %s...%s\n", ansiDim, group.source, ansiReset)

		skillArgs := group.skills
		if len(skillArgs) == 1 {
			skillArgs = []string{skillArgs[0]}
		} else {
			// Pass skills as filter
			skillArgs = group.skills
		}

		opts := AddOptions{
			Project: true,
			Yes:     yes || true,
			Skills:  skillArgs,
		}

		src := group.source
		if group.ref != "" && !strings.Contains(src, "#") {
			src = src + "#" + group.ref
		}
		runAdd(src, opts)
	}

	fmt.Printf("%sDone.%s\n\n", ansiText, ansiReset)
}
