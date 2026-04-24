package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ─── Options ───────────────────────────────────────────────────────────────────

type AddOptions struct {
	Global    bool
	Project   bool
	Agents    []string // empty = prompt
	Skills    []string // empty = prompt; "*" = all
	ListOnly  bool
	Yes       bool // skip prompts
	Copy      bool
	All       bool // --all: skill '*', agent '*', -y
	FullDepth bool
}

func parseAddOptions(args []string) (string, AddOptions) {
	var src string
	var opts AddOptions
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--global" || a == "-g":
			opts.Global = true
		case a == "--project" || a == "-p":
			opts.Project = true
		case a == "--copy":
			opts.Copy = true
		case a == "--list" || a == "-l":
			opts.ListOnly = true
		case a == "--yes" || a == "-y":
			opts.Yes = true
		case a == "--all":
			opts.All = true
		case a == "--full-depth":
			opts.FullDepth = true
		case a == "--agent" || a == "-a":
			i++
			for i < len(args) && !strings.HasPrefix(args[i], "-") {
				opts.Agents = append(opts.Agents, args[i])
				i++
			}
			i-- // step back so loop increment doesn't skip
		case a == "--skill" || a == "-s":
			i++
			for i < len(args) && !strings.HasPrefix(args[i], "-") {
				opts.Skills = append(opts.Skills, args[i])
				i++
			}
			i--
		default:
			if !strings.HasPrefix(a, "-") && src == "" {
				src = a
			}
		}
	}
	if opts.All {
		opts.Skills = []string{"*"}
		opts.Agents = []string{"*"}
		opts.Yes = true
	}
	return src, opts
}

// ─── Main add command ──────────────────────────────────────────────────────────

func runAdd(sourceInput string, opts AddOptions) {
	cwd, _ := os.Getwd()

	if sourceInput == "" {
		fmt.Fprintf(os.Stderr, "%sError:%s Please provide a package source.\n\n", ansiText, ansiReset)
		fmt.Printf("  %s$%s skl add <package>\n\n", ansiDim, ansiReset)
		fmt.Printf("  %sExamples:%s\n", ansiDim, ansiReset)
		fmt.Printf("    skl add vercel-labs/agent-skills\n")
		fmt.Printf("    skl add https://github.com/owner/repo\n")
		os.Exit(1)
	}

	parsed := parseSource(sourceInput)

	fmt.Println()

	switch parsed.Type {
	case SourceTypeWellKnown:
		runAddWellKnown(parsed, opts, cwd)
	case SourceTypeLocal:
		runAddLocal(parsed, opts, cwd)
	default:
		runAddGitOrHub(parsed, opts, cwd, sourceInput)
	}
}

// ─── Well-known provider ───────────────────────────────────────────────────────

func runAddWellKnown(parsed ParsedSource, opts AddOptions, cwd string) {
	spin := NewSpinner("Fetching skills from " + parsed.URL + "...")
	skills, err := fetchAllWellKnownSkills(parsed.URL)
	spin.Stop("")
	if err != nil || len(skills) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo skills found at %s%s\n", ansiText, parsed.URL, ansiReset)
		os.Exit(1)
	}

	// Apply skill filter
	filtered := skills
	if len(opts.Skills) > 0 && opts.Skills[0] != "*" {
		var keep []*WellKnownSkill
		for _, s := range skills {
			for _, f := range opts.Skills {
				if strings.EqualFold(s.Name, f) || strings.EqualFold(s.InstallName, f) {
					keep = append(keep, s)
					break
				}
			}
		}
		filtered = keep
	}

	if len(filtered) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo matching skills found.%s\n", ansiText, ansiReset)
		os.Exit(1)
	}

	if opts.ListOnly {
		fmt.Printf("%sAvailable skills from %s:%s\n\n", ansiText, parsed.URL, ansiReset)
		for _, s := range filtered {
			fmt.Printf("  %s%s%s  %s%s%s\n", ansiText, s.Name, ansiReset, ansiDim, s.Description, ansiReset)
		}
		return
	}

	// Skill selection
	var selectedSkills []*WellKnownSkill
	if len(opts.Skills) > 0 && opts.Skills[0] == "*" {
		selectedSkills = filtered
	} else if len(filtered) == 1 || opts.Yes {
		selectedSkills = filtered
	} else {
		options := make([]UIOption, len(filtered))
		for i, s := range filtered {
			options[i] = UIOption{Label: s.Name, Value: s.InstallName, Hint: s.Description}
		}
		var initSel []int
		for i := range filtered {
			initSel = append(initSel, i)
		}
		indices, ok := uiMultiselect("Which skills would you like to install?", options, true, initSel, nil)
		if !ok {
			fmt.Println("Cancelled.")
			return
		}
		for _, i := range indices {
			selectedSkills = append(selectedSkills, filtered[i])
		}
	}

	global, mode, agents, ok := promptScopeAndAgents(opts, cwd)
	if !ok {
		return
	}

	sourceID := getWellKnownSourceIdentifier(parsed.URL)
	fmt.Println()

	for _, skill := range selectedSkills {
		sName := sanitizeName(skill.InstallName)
		fmt.Printf("%sInstalling %s%s%s...\n", ansiDim, ansiText, skill.Name, ansiReset)

		var failedAgents []string
		for _, agentName := range agents {
			result := installWellKnownSkillForAgent(skill, agentName, global, mode)
			if !result.Success {
				failedAgents = append(failedAgents, agentName)
			}
		}

		if len(failedAgents) == 0 {
			logSuccess(skill.Name)
		} else {
			logWarn(fmt.Sprintf("%s (failed for: %s)", skill.Name, strings.Join(failedAgents, ", ")))
		}

		// Update global lock
		ownerRepo := ""
		_ = addSkillToLock(sName, SkillLockEntry{
			Source:     sourceID,
			SourceType: string(SourceTypeWellKnown),
			SourceURL:  parsed.URL,
			PluginName: skill.InstallName,
		})

		// Update local lock if project scope
		if !global {
			_ = addSkillToLocalLock(sName, LocalSkillLockEntry{
				Source:     parsed.URL,
				SourceType: string(SourceTypeWellKnown),
			}, cwd)
		}

		_ = ownerRepo
	}

	fmt.Println()
	printInstallSummary(len(selectedSkills), global, agents, mode)
	maybeShowFindPrompt(cwd)
}

// ─── Local path ────────────────────────────────────────────────────────────────

func runAddLocal(parsed ParsedSource, opts AddOptions, cwd string) {
	localPath := parsed.LocalPath
	if !pathExists(localPath) {
		fmt.Fprintf(os.Stderr, "%sError:%s Path not found: %s\n", ansiText, ansiReset, localPath)
		os.Exit(1)
	}

	skills := discoverSkillsInDir(localPath, opts.FullDepth, "")
	if len(skills) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo skills found in %s%s\n", ansiText, localPath, ansiReset)
		os.Exit(1)
	}

	if opts.ListOnly {
		fmt.Printf("%sAvailable skills:%s\n\n", ansiText, ansiReset)
		for _, s := range skills {
			fmt.Printf("  %s%s%s  %s%s%s\n", ansiText, s.Name, ansiReset, ansiDim, s.Description, ansiReset)
		}
		return
	}

	selectedSkills := selectSkills(skills, opts)

	global, mode, agents, ok := promptScopeAndAgents(opts, cwd)
	if !ok {
		return
	}

	fmt.Println()
	installSkillsForAgents(selectedSkills, agents, global, mode, SkillLockEntry{
		Source:     localPath,
		SourceType: string(SourceTypeLocal),
		SourceURL:  localPath,
	}, cwd, false)
	maybeShowFindPrompt(cwd)
}

// ─── GitHub / GitLab / git ─────────────────────────────────────────────────────

func runAddGitOrHub(parsed ParsedSource, opts AddOptions, cwd, sourceInput string) {
	ownerRepo := getOwnerRepo(parsed)
	token := getGitHubToken()

	// Try blob fast-install (vercel/vercel-labs only)
	if parsed.Type == SourceTypeGitHub && ownerRepo != "" {
		blobOpts := struct {
			Subpath         string
			SkillFilter     string
			Ref             string
			Token           string
			IncludeInternal bool
		}{
			Subpath:     parsed.Subpath,
			SkillFilter: skillFilterFromOpts(opts, parsed),
			Ref:         parsed.Ref,
			Token:       token,
		}

		spin := NewSpinner("Fetching skills...")
		blobResult, _ := tryBlobInstall(ownerRepo, blobOpts)
		spin.Stop("")

		if blobResult != nil && len(blobResult.Skills) > 0 {
			runAddBlob(blobResult, parsed, opts, cwd, ownerRepo, sourceInput)
			return
		}
	}

	// OSV advisory check (async, don't block)
	osvCh := make(chan *OSVResult, 1)
	if ownerRepo != "" && parsed.Type == SourceTypeGitHub {
		go func() {
			osvCh <- fetchOSVAdvisories(ownerRepo, 5000)
		}()
	} else {
		osvCh <- nil
	}

	// Clone repo
	ref := parsed.Ref
	spin := NewSpinner("Cloning " + parsed.URL + "...")
	tmpDir, err := cloneRepo(parsed.URL, ref)
	spin.Stop("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %s\n", ansiText, ansiReset, err.Error())
		os.Exit(1)
	}
	defer cleanupTempDir(tmpDir)

	searchRoot := tmpDir
	if parsed.Subpath != "" {
		searchRoot = filepath.Join(tmpDir, parsed.Subpath)
		if !pathExists(searchRoot) {
			fmt.Fprintf(os.Stderr, "%sSubpath not found:%s %s\n", ansiText, ansiReset, parsed.Subpath)
			os.Exit(1)
		}
	}

	skillFilter := skillFilterFromOpts(opts, parsed)
	skills := discoverSkillsInDir(searchRoot, opts.FullDepth, skillFilter)
	if len(skills) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo skills found in %s%s\n", ansiText, parsed.URL, ansiReset)
		os.Exit(1)
	}

	if opts.ListOnly {
		fmt.Printf("%sAvailable skills:%s\n\n", ansiText, ansiReset)
		for _, s := range skills {
			fmt.Printf("  %s%s%s  %s%s%s\n", ansiText, s.Name, ansiReset, ansiDim, s.Description, ansiReset)
		}
		return
	}

	selectedSkills := selectSkills(skills, opts)

	global, mode, agents, ok := promptScopeAndAgents(opts, cwd)
	if !ok {
		return
	}

	// Show OSV advisories if any
	osv := <-osvCh
	if osv != nil && osv.Count > 0 {
		showOSVAdvisories(osv, ownerRepo)
	}

	fmt.Println()

	// Build lock entry
	lockRef := ref
	if lockRef == "" {
		lockRef = "main"
	}
	baseLockEntry := SkillLockEntry{
		Source:     sourceInput,
		SourceType: string(parsed.Type),
		SourceURL:  parsed.URL,
		Ref:        lockRef,
	}

	installSkillsForAgents(selectedSkills, agents, global, mode, baseLockEntry, cwd, ownerRepo != "" && parsed.Type == SourceTypeGitHub)

	maybeShowFindPrompt(cwd)
}

// ─── Blob fast install ─────────────────────────────────────────────────────────

func runAddBlob(result *BlobInstallResult, parsed ParsedSource, opts AddOptions, cwd, ownerRepo, sourceInput string) {
	skills := result.Skills

	// Apply skill filter
	skillFilter := skillFilterFromOpts(opts, parsed)
	if skillFilter != "" && skillFilter != "*" {
		var keep []*BlobSkill
		for _, s := range skills {
			if strings.EqualFold(s.Name, skillFilter) || strings.EqualFold(toSkillSlug(s.Name), skillFilter) {
				keep = append(keep, s)
			}
		}
		skills = keep
	}

	if len(skills) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo matching skills found.%s\n", ansiText, ansiReset)
		os.Exit(1)
	}

	if opts.ListOnly {
		fmt.Printf("%sAvailable skills:%s\n\n", ansiText, ansiReset)
		for _, s := range skills {
			fmt.Printf("  %s%s%s  %s%s%s\n", ansiText, s.Name, ansiReset, ansiDim, s.Description, ansiReset)
		}
		return
	}

	// Skill selection
	var selectedBlob []*BlobSkill
	if (len(opts.Skills) > 0 && opts.Skills[0] == "*") || opts.Yes || len(skills) == 1 {
		selectedBlob = skills
	} else {
		options := make([]UIOption, len(skills))
		for i, s := range skills {
			options[i] = UIOption{Label: s.Name, Value: toSkillSlug(s.Name), Hint: s.Description}
		}
		var initSel []int
		for i := range skills {
			initSel = append(initSel, i)
		}
		indices, ok := uiMultiselect("Which skills would you like to install?", options, true, initSel, nil)
		if !ok {
			fmt.Println("Cancelled.")
			return
		}
		for _, i := range indices {
			selectedBlob = append(selectedBlob, skills[i])
		}
	}

	global, mode, agents, ok := promptScopeAndAgents(opts, cwd)
	if !ok {
		return
	}

	// OSV check (non-blocking)
	osv := <-func() chan *OSVResult {
		ch := make(chan *OSVResult, 1)
		go func() { ch <- fetchOSVAdvisories(ownerRepo, 5000) }()
		return ch
	}()
	if osv != nil && osv.Count > 0 {
		showOSVAdvisories(osv, ownerRepo)
	}

	fmt.Println()
	ref := result.Tree.Branch
	if parsed.Ref != "" {
		ref = parsed.Ref
	}

	for _, bSkill := range selectedBlob {
		sName := sanitizeName(bSkill.Name)
		fmt.Printf("%sInstalling %s%s%s...\n", ansiDim, ansiText, bSkill.Name, ansiReset)

		files := make([]struct{ Path, Contents string }, len(bSkill.Files))
		for i, f := range bSkill.Files {
			files[i] = struct{ Path, Contents string }{f.Path, f.Contents}
		}

		var failedAgents []string
		for _, agentName := range agents {
			result := installSkillFilesForAgent(sName, files, agentName, global, mode)
			if !result.Success {
				failedAgents = append(failedAgents, agentName)
			}
		}

		if len(failedAgents) == 0 {
			logSuccess(bSkill.Name)
		} else {
			logWarn(fmt.Sprintf("%s (failed for: %s)", bSkill.Name, strings.Join(failedAgents, ", ")))
		}

		// Compute folder hash from tree
		folderHash := ""
		if result.Tree != nil {
			folderHash = getSkillFolderHashFromTree(result.Tree, bSkill.RepoPath)
		}

		_ = addSkillToLock(sName, SkillLockEntry{
			Source:          sourceInput,
			SourceType:      string(SourceTypeGitHub),
			SourceURL:       parsed.URL,
			Ref:             ref,
			SkillPath:       bSkill.RepoPath,
			SkillFolderHash: folderHash,
			PluginName:      sName,
		})
		if !global {
			_ = addSkillToLocalLock(sName, LocalSkillLockEntry{
				Source:     sourceInput,
				Ref:        ref,
				SourceType: string(SourceTypeGitHub),
			}, cwd)
		}
	}

	fmt.Println()
	printInstallSummary(len(selectedBlob), global, agents, mode)
	maybeShowFindPrompt(cwd)
}

// ─── Shared install logic ──────────────────────────────────────────────────────

func installSkillsForAgents(skills []*Skill, agents []string, global bool, mode InstallMode, baseLockEntry SkillLockEntry, cwd string, computeHash bool) {
	for _, skill := range skills {
		sName := sanitizeName(skill.Name)
		fmt.Printf("%sInstalling %s%s%s...\n", ansiDim, ansiText, skill.Name, ansiReset)

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

		lockEntry := baseLockEntry
		lockEntry.PluginName = sName
		if computeHash && skill.Path != "" {
			if hash, err := computeSkillFolderHash(skill.Path); err == nil {
				lockEntry.SkillFolderHash = hash
			}
		}

		_ = addSkillToLock(sName, lockEntry)
		if !global {
			_ = addSkillToLocalLock(sName, LocalSkillLockEntry{
				Source:     baseLockEntry.Source,
				Ref:        baseLockEntry.Ref,
				SourceType: baseLockEntry.SourceType,
			}, cwd)
		}
	}

	fmt.Println()
	printInstallSummary(len(skills), global, agents, mode)
}

// ─── Skill discovery ───────────────────────────────────────────────────────────

func discoverSkillsInDir(dir string, fullDepth bool, skillFilter string) []*Skill {
	maxDepth := 5
	if fullDepth {
		maxDepth = 20
	}

	allFound := findSkillDirs(dir, 0, maxDepth)

	var skills []*Skill
	seen := map[string]bool{}
	for _, skillMd := range allFound {
		s, err := parseSkillMd(filepath.Join(skillMd, "SKILL.md"), false)
		if err != nil || s == nil {
			continue
		}
		if seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		skills = append(skills, s)
	}

	if skillFilter != "" && skillFilter != "*" {
		var filtered []*Skill
		for _, s := range skills {
			if strings.EqualFold(s.Name, skillFilter) || strings.EqualFold(sanitizeName(s.Name), sanitizeName(skillFilter)) {
				filtered = append(filtered, s)
			}
		}
		skills = filtered
	}

	return skills
}

// ─── Prompt helpers ────────────────────────────────────────────────────────────

func selectSkills(skills []*Skill, opts AddOptions) []*Skill {
	if (len(opts.Skills) > 0 && opts.Skills[0] == "*") || opts.Yes || len(skills) == 1 {
		if len(opts.Skills) > 0 && opts.Skills[0] != "*" {
			// Filter to only named skills
			var filtered []*Skill
			for _, s := range skills {
				for _, f := range opts.Skills {
					if strings.EqualFold(s.Name, f) || strings.EqualFold(sanitizeName(s.Name), sanitizeName(f)) {
						filtered = append(filtered, s)
						break
					}
				}
			}
			return filtered
		}
		return skills
	}

	if len(opts.Skills) > 0 {
		var filtered []*Skill
		for _, s := range skills {
			for _, f := range opts.Skills {
				if strings.EqualFold(s.Name, f) || strings.EqualFold(sanitizeName(s.Name), sanitizeName(f)) {
					filtered = append(filtered, s)
					break
				}
			}
		}
		return filtered
	}

	options := make([]UIOption, len(skills))
	for i, s := range skills {
		options[i] = UIOption{Label: s.Name, Value: sanitizeName(s.Name), Hint: s.Description}
	}
	var initSel []int
	for i := range skills {
		initSel = append(initSel, i)
	}
	indices, ok := uiMultiselect("Which skills would you like to install?", options, true, initSel, nil)
	if !ok {
		fmt.Println("Cancelled.")
		os.Exit(0)
	}
	var selected []*Skill
	for _, i := range indices {
		selected = append(selected, skills[i])
	}
	return selected
}

// promptScopeAndAgents asks for global/project scope and agent selection.
// Returns (global bool, mode InstallMode, agents []string, ok bool).
func promptScopeAndAgents(opts AddOptions, cwd string) (bool, InstallMode, []string, bool) {
	// Determine scope
	global := opts.Global
	if !global && !opts.Project && !opts.Yes {
		idx, ok := uiSelect("Install scope?", []UIOption{
			{Label: "Project", Hint: "installs for this project only"},
			{Label: "Global", Hint: "installs for your user account"},
		})
		if !ok {
			return false, "", nil, false
		}
		global = idx == 1
	}

	// Determine install mode
	mode := InstallModeSymlink
	if opts.Copy {
		mode = InstallModeCopy
	}

	// Determine agents
	agents, ok := promptAgents(opts, global, cwd)
	if !ok {
		return false, "", nil, false
	}

	return global, mode, agents, true
}

// promptAgents returns the list of agents to install to.
func promptAgents(opts AddOptions, global bool, cwd string) ([]string, bool) {
	if len(opts.Agents) > 0 && opts.Agents[0] == "*" {
		var all []string
		for name := range allAgents {
			if global && allAgents[name].GlobalSkillsDir == "" {
				continue
			}
			all = append(all, name)
		}
		sort.Strings(all)
		return all, true
	}

	if len(opts.Agents) > 0 {
		var validated []string
		for _, a := range opts.Agents {
			if allAgents[a] != nil {
				validated = append(validated, a)
			} else {
				logWarn("Unknown agent: " + a)
			}
		}
		if len(validated) == 0 {
			fmt.Fprintf(os.Stderr, "%sNo valid agents specified.%s\n", ansiText, ansiReset)
			return nil, false
		}
		return validated, true
	}

	// Detect installed agents
	detected := detectInstalledAgents()

	// Universal agents (always shown as locked in UI)
	universalAgents := getUniversalAgents()
	nonUniversal := getNonUniversalAgents()

	// Filter to installed non-universal agents
	var installedNonUniversal []string
	for _, a := range nonUniversal {
		for _, d := range detected {
			if d == a {
				installedNonUniversal = append(installedNonUniversal, a)
				break
			}
		}
	}

	// Build UI options from non-universal installed agents
	var options []UIOption
	for _, a := range installedNonUniversal {
		cfg := allAgents[a]
		if cfg == nil {
			continue
		}
		if global && cfg.GlobalSkillsDir == "" {
			continue
		}
		options = append(options, UIOption{Label: cfg.DisplayName, Value: a})
	}

	// Add non-installed agents at the bottom
	for name, cfg := range allAgents {
		if isUniversalAgent(name) {
			continue
		}
		if global && cfg.GlobalSkillsDir == "" {
			continue
		}
		alreadyIn := false
		for _, opt := range options {
			if opt.Value == name {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn {
			options = append(options, UIOption{Label: cfg.DisplayName, Value: name, Hint: "not detected"})
		}
	}

	// Build locked options for universal agents
	var lockedOptions []UIOption
	for _, a := range universalAgents {
		cfg := allAgents[a]
		if cfg == nil {
			continue
		}
		if global && cfg.GlobalSkillsDir == "" {
			continue
		}
		lockedOptions = append(lockedOptions, UIOption{Label: cfg.DisplayName, Value: a})
	}

	// Load last selected agents as initial selection
	lastSelected := getLastSelectedAgents()
	lastSelectedSet := map[string]bool{}
	for _, a := range lastSelected {
		lastSelectedSet[a] = true
	}

	// Initial selection: last selected non-universal agents
	var initSel []int
	for i, opt := range options {
		if lastSelectedSet[opt.Value] {
			initSel = append(initSel, i)
		}
	}
	// If nothing was previously selected, select detected agents by default
	if len(initSel) == 0 {
		for i, opt := range options {
			for _, d := range installedNonUniversal {
				if opt.Value == d {
					initSel = append(initSel, i)
					break
				}
			}
		}
	}

	message := "Which agents would you like to install to?"
	if opts.Yes {
		// Return all detected + universal
		var result []string
		for _, lo := range lockedOptions {
			result = append(result, lo.Value)
		}
		for _, opt := range options {
			alreadyDetected := false
			for _, d := range detected {
				if d == opt.Value {
					alreadyDetected = true
					break
				}
			}
			if alreadyDetected {
				result = append(result, opt.Value)
			}
		}
		return result, true
	}

	if len(options) == 0 && len(lockedOptions) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo agents available.%s\n", ansiText, ansiReset)
		return nil, false
	}

	// If only universal agents, skip prompt
	if len(options) == 0 {
		var result []string
		for _, lo := range lockedOptions {
			result = append(result, lo.Value)
		}
		return result, true
	}

	selectedIndices, ok := uiSearchMultiselect(message, options, lockedOptions, initSel)
	if !ok {
		return nil, false
	}

	var result []string
	for _, lo := range lockedOptions {
		result = append(result, lo.Value)
	}
	for _, i := range selectedIndices {
		result = append(result, options[i].Value)
	}

	// Save selection for next time
	_ = saveSelectedAgents(result)

	return result, true
}

// ─── Post-install helpers ──────────────────────────────────────────────────────

func printInstallSummary(count int, global bool, agents []string, mode InstallMode) {
	scope := "project"
	if global {
		scope = "global"
	}
	noun := "skill"
	if count != 1 {
		noun = "skills"
	}
	fmt.Printf("%s✓ Installed %d %s (%s scope, %s mode)%s\n", ansiText, count, noun, scope, string(mode), ansiReset)
	if len(agents) > 0 {
		var displayNames []string
		for _, a := range agents {
			if cfg := allAgents[a]; cfg != nil {
				displayNames = append(displayNames, cfg.DisplayName)
			} else {
				displayNames = append(displayNames, a)
			}
		}
		fmt.Printf("%s  Agents: %s%s\n", ansiDim, strings.Join(displayNames, ", "), ansiReset)
	}
	fmt.Println()
}

func maybeShowFindPrompt(cwd string) {
	if isPromptDismissed("findSkillsPrompt") {
		return
	}
	fmt.Printf("%sTip:%s Run %sskl find%s to discover more skills.\n", ansiDim, ansiReset, ansiText, ansiReset)
	_ = dismissPrompt("findSkillsPrompt")
}

func showOSVAdvisories(osv *OSVResult, ownerRepo string) {
	if osv == nil || osv.Count == 0 {
		return
	}
	fmt.Printf("\n%s⚠ Security advisories for %s:%s\n", ansiText, ownerRepo, ansiReset)
	for _, a := range osv.Advisories {
		fmt.Printf("  %s[%s]%s %s%s%s  %s%s%s\n",
			ansiDim, string(a.Severity), ansiReset,
			ansiText, a.ID, ansiReset,
			ansiDim, a.Summary, ansiReset)
	}
	fmt.Println()
}

func skillFilterFromOpts(opts AddOptions, parsed ParsedSource) string {
	if len(opts.Skills) > 0 {
		return opts.Skills[0]
	}
	return parsed.SkillFilter
}
