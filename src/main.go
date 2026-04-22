package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const Version = "1.8.3"

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[38;5;102m"
	ansiText   = "\x1b[38;5;145m"
	ansiCyan   = "\x1b[36m"
	ansiYellow = "\x1b[33m"
	ansiGreen  = "\x1b[32m"
	ansiRed    = "\x1b[31m"
)

var logoLines = []string{
	"███████╗██╗  ██╗██╗     ",
	"██╔════╝██║ ██╔╝██║     ",
	"███████╗█████╔╝ ██║     ",
	"╚════██║██╔═██╗ ██║     ",
	"███████║██║  ██╗███████╗",
	"╚══════╝╚═╝  ╚═╝╚══════╝",
}

var grays = []string{
	"\x1b[38;5;250m",
	"\x1b[38;5;248m",
	"\x1b[38;5;245m",
	"\x1b[38;5;243m",
	"\x1b[38;5;240m",
	"\x1b[38;5;238m",
}

func showLogo() {
	fmt.Println()
	for i, line := range logoLines {
		fmt.Printf("%s%s%s\n", grays[i], line, ansiReset)
	}
}

func showBanner() {
	showLogo()
	fmt.Println()
	fmt.Printf("%sThe agent skill management CLI%s\n", ansiDim, ansiReset)
	fmt.Printf("%sNo telemetry. Fully open source.%s\n", ansiDim, ansiReset)
	fmt.Println()
	fmt.Printf("  %s$%s %sskl add %s<package>%s           %sAdd a new skill%s\n", ansiDim, ansiReset, ansiText, ansiDim, ansiReset, ansiDim, ansiReset)
	fmt.Printf("  %s$%s %sskl remove%s                  %sRemove installed skills%s\n", ansiDim, ansiReset, ansiText, ansiReset, ansiDim, ansiReset)
	fmt.Printf("  %s$%s %sskl list%s                    %sList installed skills%s\n", ansiDim, ansiReset, ansiText, ansiReset, ansiDim, ansiReset)
	fmt.Printf("  %s$%s %sskl find %s[query]%s            %sSearch for skills%s\n", ansiDim, ansiReset, ansiText, ansiDim, ansiReset, ansiDim, ansiReset)
	fmt.Println()
	fmt.Printf("  %s$%s %sskl update%s                  %sUpdate installed skills%s\n", ansiDim, ansiReset, ansiText, ansiReset, ansiDim, ansiReset)
	fmt.Println()
	fmt.Printf("  %s$%s %sskl experimental_install%s    %sRestore from skills-lock.json%s\n", ansiDim, ansiReset, ansiText, ansiReset, ansiDim, ansiReset)
	fmt.Printf("  %s$%s %sskl init %s[name]%s             %sCreate a new skill%s\n", ansiDim, ansiReset, ansiText, ansiDim, ansiReset, ansiDim, ansiReset)
	fmt.Printf("  %s$%s %sskl experimental_sync%s       %sSync skills from node_modules%s\n", ansiDim, ansiReset, ansiText, ansiReset, ansiDim, ansiReset)
	fmt.Println()
	fmt.Printf("%stry:%s skl add vercel-labs/agent-skills\n", ansiDim, ansiReset)
	fmt.Println()
	fmt.Printf("Discover more skills at %shttps://skl.sh/%s\n", ansiText, ansiReset)
	fmt.Println()
}

func showHelp() {
	fmt.Printf(`
%sUsage:%s skl <command> [options]

%sManage Skills:%s
  add <package>        Add a skill package (alias: a, install, i)
                       e.g. vercel-labs/agent-skills
                            https://github.com/vercel-labs/agent-skills
  remove [skills]      Remove installed skills (alias: rm, r)
  list, ls             List installed skills
  find [query]         Search for skills interactively (alias: search, f, s)

%sUpdates:%s
  update [skills...]   Update skills to latest versions (alias: upgrade, check)
  update-cli           Check for CLI updates and self-update the binary

%sUpdate Options:%s
  -g, --global           Update global skills only
  -p, --project          Update project skills only
  -y, --yes              Skip scope prompt

%sProject:%s
  experimental_install Restore skills from skills-lock.json
  init [name]          Initialize a skill (creates <name>/SKILL.md or ./SKILL.md)
  experimental_sync    Sync skills from node_modules into agent directories

%sInstall Options:%s
  -a, --agent <agents>   Specify agents to install to (skips agent selection prompt)
  -y, --yes              Skip agent selection prompt

%sAdd Options:%s
  -p, --project          Force project-scope install
  -g, --global           Install skill globally (user-level)
  -a, --agent <agents>   Specify agents to install to (use '*' for all agents)
  -s, --skill <skills>   Specify skill names to install (use '*' for all skills)
  -l, --list             List available skills without installing
  -y, --yes              Skip confirmation prompts
  --copy                 Copy files instead of symlinking
  --all                  Shorthand for --skill '*' --agent '*' -y
  --full-depth           Search all subdirectories

%sRemove Options:%s
  -g, --global           Remove from global scope
  -a, --agent <agents>   Remove from specific agents
  -s, --skill <skills>   Specify skills to remove
  -y, --yes              Skip confirmation prompts
  --all                  Shorthand for --skill '*' --agent '*' -y

%sList Options:%s
  -g, --global           List global skills (default: project)
  -a, --agent <agents>   Filter by specific agents
  --json                 Output as JSON

%sOptions:%s
  --help, -h        Show this help message
  --version, -v     Show version number

%sExamples:%s
  %s$%s skl add vercel-labs/agent-skills
  %s$%s skl add vercel-labs/agent-skills -g
  %s$%s skl add vercel-labs/agent-skills --agent claude-code cursor
  %s$%s skl remove
  %s$%s skl list
  %s$%s skl ls -g
  %s$%s skl find typescript
  %s$%s skl update

Discover more skills at %shttps://skl.sh/%s
`,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiDim, ansiReset,
		ansiDim, ansiReset,
		ansiDim, ansiReset,
		ansiDim, ansiReset,
		ansiDim, ansiReset,
		ansiDim, ansiReset,
		ansiDim, ansiReset,
		ansiDim, ansiReset,
		ansiText, ansiReset,
	)
}

func runInitCmd(args []string) {
	cwd, _ := os.Getwd()
	var skillName string
	hasName := len(args) > 0 && args[0] != ""
	if hasName {
		skillName = args[0]
	} else {
		skillName = filepath.Base(cwd)
	}

	var skillDir string
	if hasName {
		skillDir = filepath.Join(cwd, skillName)
	} else {
		skillDir = cwd
	}
	skillFile := filepath.Join(skillDir, "SKILL.md")

	var displayPath string
	if hasName {
		displayPath = skillName + "/SKILL.md"
	} else {
		displayPath = "SKILL.md"
	}

	if _, err := os.Stat(skillFile); err == nil {
		fmt.Printf("%sSkill already exists at %s%s%s\n", ansiText, ansiDim, displayPath, ansiReset)
		return
	}

	if hasName {
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}
	}

	content := fmt.Sprintf(`---
name: %s
description: A brief description of what this skill does
---

# %s

Instructions for the agent to follow when this skill is activated.

## When to use

Describe when this skill should be used.

## Instructions

1. First step
2. Second step
3. Additional steps as needed
`, skillName, skillName)

	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%sInitialized skill: %s%s%s\n", ansiText, ansiDim, skillName, ansiReset)
	fmt.Println()
	fmt.Printf("%sCreated:%s\n", ansiDim, ansiReset)
	fmt.Printf("  %s\n", displayPath)
	fmt.Println()
	fmt.Printf("%sNext steps:%s\n", ansiDim, ansiReset)
	fmt.Printf("  1. Edit %s%s%s to define your skill instructions\n", ansiText, displayPath, ansiReset)
	fmt.Printf("  2. Update the %sname%s and %sdescription%s in the frontmatter\n", ansiText, ansiReset, ansiText, ansiReset)
	fmt.Println()
	fmt.Printf("%sPublishing:%s\n", ansiDim, ansiReset)
	fmt.Printf("  %sGitHub:%s  Push to a repo, then %sskl add <owner>/<repo>%s\n", ansiDim, ansiReset, ansiText, ansiReset)
	fmt.Println()
	fmt.Printf("Browse existing skills for inspiration at %shttps://skl.sh/%s\n", ansiText, ansiReset)
	fmt.Println()
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		showBanner()
		return
	}

	command := args[0]
	rest := args[1:]

	switch command {
	case "find", "search", "f", "s":
		showLogo()
		fmt.Println()
		runFind(rest)

	case "init":
		showLogo()
		fmt.Println()
		runInitCmd(rest)

	case "experimental_install":
		showLogo()
		runInstallFromLock(rest)

	case "i", "install", "a", "add":
		showLogo()
		src, opts := parseAddOptions(rest)
		runAdd(src, opts)

	case "remove", "rm", "r":
		if contains(rest, "--help") || contains(rest, "-h") {
			showRemoveHelp()
			return
		}
		skills, opts := parseRemoveOptions(rest)
		runRemove(skills, opts)

	case "experimental_sync":
		showLogo()
		opts := parseSyncOptions(rest)
		runSync(rest, opts)

	case "list", "ls":
		runList(rest)

	case "check", "update", "upgrade":
		runUpdate(rest)

	case "update-cli", "self-update", "upgrade-cli":
		runSelfUpdate(Version)

	case "--help", "-h", "help":
		showHelp()

	case "--version", "-v", "version":
		fmt.Println(Version)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Printf("Run %sskl --help%s for usage.\n", ansiBold, ansiReset)
		os.Exit(1)
	}
}

func showRemoveHelp() {
	fmt.Printf(`
%sUsage:%s skl remove [skills...] [options]

%sDescription:%s
  Remove installed skills from agents. If no skill names are provided,
  an interactive selection menu will be shown.

%sArguments:%s
  skills            Optional skill names to remove (space-separated)

%sOptions:%s
  -g, --global       Remove from global scope (~/) instead of project scope
  -a, --agent        Remove from specific agents (use '*' for all agents)
  -s, --skill        Specify skills to remove (use '*' for all skills)
  -y, --yes          Skip confirmation prompts
  --all              Shorthand for --skill '*' --agent '*' -y

%sExamples:%s
  %s$%s skl remove                           %s# interactive selection%s
  %s$%s skl remove my-skill                  %s# remove specific skill%s
  %s$%s skl remove skill1 skill2 -y          %s# remove multiple skills%s
  %s$%s skl remove --global my-skill         %s# remove from global scope%s
  %s$%s skl remove --all                     %s# remove all skills%s

Discover more skills at %shttps://skl.sh/%s
`,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiBold, ansiReset,
		ansiDim, ansiReset, ansiDim, ansiReset,
		ansiDim, ansiReset, ansiDim, ansiReset,
		ansiDim, ansiReset, ansiDim, ansiReset,
		ansiDim, ansiReset, ansiDim, ansiReset,
		ansiDim, ansiReset, ansiDim, ansiReset,
		ansiText, ansiReset,
	)
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
