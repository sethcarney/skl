package main

import (
	"fmt"
	"os"
	"strings"
)

type UpdateOptions struct {
	Global  bool
	Project bool
	Yes     bool
}

func runUpdate(args []string) {
	var opts UpdateOptions
	var skillFilter []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--global" || a == "-g":
			opts.Global = true
		case a == "--project" || a == "-p":
			opts.Project = true
		case a == "--yes" || a == "-y":
			opts.Yes = true
		default:
			if !strings.HasPrefix(a, "-") {
				skillFilter = append(skillFilter, a)
			}
		}
	}

	// Determine scope
	global := opts.Global
	project := opts.Project
	if !global && !project && !opts.Yes {
		idx, ok := uiSelect("Update which scope?", []UIOption{
			{Label: "Both", Hint: "project and global"},
			{Label: "Project"},
			{Label: "Global"},
		})
		if !ok {
			return
		}
		switch idx {
		case 0:
			global = true
			project = true
		case 1:
			project = true
		case 2:
			global = true
		}
	} else if !global && !project {
		global = true
		project = true
	}

	updated := 0
	skipped := 0
	failed := 0

	// Check global skills from lock
	if global {
		lock := readSkillLock()
		for sName, entry := range lock.Skills {
			if len(skillFilter) > 0 {
				found := false
				for _, f := range skillFilter {
					if strings.EqualFold(sName, f) || strings.EqualFold(f, entry.PluginName) {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			if entry.SourceType != string(SourceTypeGitHub) && entry.SourceType != string(SourceTypeGitLab) && entry.SourceType != string(SourceTypeGit) {
				skipped++
				continue
			}

			fmt.Printf("%sChecking %s...%s\n", ansiDim, sName, ansiReset)
			isUpToDate, err := checkSkillUpToDate(sName, entry)
			if err != nil {
				logWarn(fmt.Sprintf("Could not check %s: %v", sName, err))
				skipped++
				continue
			}
			if isUpToDate {
				logInfo(sName + " is up to date")
				skipped++
				continue
			}

			// Re-run add to update
			addOpts := AddOptions{
				Global: true,
				Yes:    true,
				Skills: []string{entry.PluginName},
			}
			runAdd(entry.Source, addOpts)
			updated++
		}
	}

	// Check project skills from local lock
	if project {
		cwd, _ := os.Getwd()
		localLock := readLocalLock(cwd)
		for sName, entry := range localLock.Skills {
			if len(skillFilter) > 0 {
				found := false
				for _, f := range skillFilter {
					if strings.EqualFold(sName, f) {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			if entry.SourceType != string(SourceTypeGitHub) && entry.SourceType != string(SourceTypeGitLab) && entry.SourceType != string(SourceTypeGit) {
				skipped++
				continue
			}

			fmt.Printf("%sChecking %s...%s\n", ansiDim, sName, ansiReset)

			addOpts := AddOptions{
				Project: true,
				Yes:     true,
				Skills:  []string{sName},
			}
			src := entry.Source
			if entry.Ref != "" && !strings.Contains(src, "#") {
				src = src + "#" + entry.Ref
			}
			runAdd(src, addOpts)
			updated++
		}
	}

	fmt.Println()
	if updated == 0 && skipped == 0 && failed == 0 {
		fmt.Printf("%sNo skills to update.%s\n", ansiDim, ansiReset)
		return
	}
	fmt.Printf("%sUpdate complete:%s %d updated, %d already up to date", ansiText, ansiReset, updated, skipped)
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()
	fmt.Println()
}

func checkSkillUpToDate(skillName string, entry SkillLockEntry) (bool, error) {
	if entry.SkillFolderHash == "" || entry.SkillPath == "" {
		return false, nil
	}
	ownerRepo := ""
	parsed := parseSource(entry.Source)
	ownerRepo = getOwnerRepo(parsed)
	if ownerRepo == "" {
		return false, nil
	}

	token := getGitHubToken()
	ref := entry.Ref
	latestHash, err := fetchSkillFolderHash(ownerRepo, entry.SkillPath, token, &ref)
	if err != nil {
		return false, err
	}
	return latestHash == entry.SkillFolderHash, nil
}
