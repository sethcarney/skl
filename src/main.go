package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const Version = "0.0.8"

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[38;5;243m"
	ansiText   = "\x1b[38;5;159m"
	ansiCyan   = "\x1b[36m"
	ansiYellow = "\x1b[33m"
	ansiGreen  = "\x1b[32m"
	ansiRed    = "\x1b[31m"
)

func showLogo() {
	fmt.Printf("\n%s%sskl%s %s%s%s\n\n", ansiBold, ansiText, ansiReset, ansiDim, Version, ansiReset)
}

// multiValueFlags are flags that accept multiple space-separated values after a
// single flag instance (-a claude cursor) in addition to the repeated-flag form
// (-a claude -a cursor). Both styles are supported.
var multiValueFlags = map[string]bool{
	"agent": true, "a": true,
	"skill": true, "s": true,
}

// normalizeMultiFlags rewrites space-separated multi-value flags into the
// repeated-flag form that cobra/pflag expects.
// e.g. ["-a", "claude", "cursor"] → ["-a", "claude", "-a", "cursor"]
func normalizeMultiFlags(args []string) []string {
	result := make([]string, 0, len(args))
	i := 0
	for i < len(args) {
		arg := args[i]
		i++
		result = append(result, arg)

		var flagName string
		switch {
		case strings.HasPrefix(arg, "--") && !strings.Contains(arg, "="):
			flagName = arg[2:]
		case len(arg) == 2 && arg[0] == '-' && arg[1] != '-':
			flagName = string(arg[1])
		}

		if !multiValueFlags[flagName] {
			continue
		}
		// consume first value
		if i >= len(args) || strings.HasPrefix(args[i], "-") {
			continue
		}
		result = append(result, args[i])
		i++
		// expand extra space-separated values into repeated flag+value pairs
		for i < len(args) && !strings.HasPrefix(args[i], "-") {
			result = append(result, arg, args[i])
			i++
		}
	}
	return result
}

func main() {
	root := &cobra.Command{
		Use:           "skl",
		Short:         "The agent skill management CLI",
		Long:          fmt.Sprintf("%s%sskl%s — The agent skill management CLI. No telemetry · Fully open source.", ansiBold, ansiText, ansiReset),
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println()
			fmt.Printf("%s%sskl%s\n", ansiBold, ansiText, ansiReset)
			fmt.Printf("%sThe agent skill management CLI. No telemetry · Fully open source. (%s)%s\n", ansiDim, Version, ansiReset)
			fmt.Println()
			_ = cmd.Help()
		},
	}

	root.SetVersionTemplate(fmt.Sprintf("%s%sskl%s %s\n", ansiBold, ansiText, ansiReset, Version))

	root.AddCommand(
		buildAddCmd(),
		buildRemoveCmd(),
		buildListCmd(),
		buildFindCmd(),
		buildUpdateCmd(),
		buildUpgradeCmd(),
		buildInitCmd(),
		buildInstallFromLockCmd(),
		buildSyncCmd(),
		buildCompletionCmd(root),
	)

	root.SetArgs(normalizeMultiFlags(os.Args[1:]))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ─── add ───────────────────────────────────────────────────────────────────────

func buildAddCmd() *cobra.Command {
	var opts AddOptions

	cmd := &cobra.Command{
		Use:     "add <package>",
		Short:   "Add a skill from GitHub or URL",
		Aliases: []string{"a", "install", "i"},
		Long: fmt.Sprintf(`Add a skill package from GitHub, a URL, or a local path.

The --agent (-a) and --skill (-s) flags accept multiple values. You can
pass them space-separated after the flag or repeat the flag for each value
— both styles are equivalent:

  skl add owner/repo -a claude-code cursor
  skl add owner/repo -a claude-code -a cursor

%sExamples:%s
  skl add vercel-labs/agent-skills
  skl add vercel-labs/agent-skills -g
  skl add vercel-labs/agent-skills -a claude-code cursor
  skl add vercel-labs/agent-skills --agent claude-code --agent cursor
  skl add https://github.com/owner/repo
  skl add ./my-local-skill`, ansiBold, ansiReset),
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			showLogo()
			src := ""
			if len(args) > 0 {
				src = args[0]
			}
			if opts.All {
				opts.Skills = []string{"*"}
				opts.Agents = []string{"*"}
				opts.Yes = true
			}
			runAdd(src, opts)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&opts.Global, "global", "g", false, "Install skill globally (user-level)")
	f.BoolVarP(&opts.Project, "project", "p", false, "Force project-scope install")
	f.StringArrayVarP(&opts.Agents, "agent", "a", nil, "Agents to install to (repeatable, use '*' for all)")
	f.StringArrayVarP(&opts.Skills, "skill", "s", nil, "Skill names to install (repeatable, use '*' for all)")
	f.BoolVarP(&opts.ListOnly, "list", "l", false, "List available skills without installing")
	f.BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompts")
	f.BoolVar(&opts.Copy, "copy", false, "Copy files instead of symlinking")
	f.BoolVar(&opts.All, "all", false, "Shorthand for --skill '*' --agent '*' -y")
	f.BoolVar(&opts.FullDepth, "full-depth", false, "Search all subdirectories")

	return cmd
}

// ─── remove ────────────────────────────────────────────────────────────────────

func buildRemoveCmd() *cobra.Command {
	var opts RemoveOptions

	cmd := &cobra.Command{
		Use:     "remove [skills...]",
		Short:   "Remove installed skills",
		Aliases: []string{"rm", "r"},
		Long: fmt.Sprintf(`Remove installed skills from agents.

If no skill names are provided an interactive selection menu is shown.

The --agent (-a) and --skill (-s) flags accept multiple values — space-
separated after the flag or repeated:

  skl remove -a claude-code cursor
  skl remove -a claude-code -a cursor

%sExamples:%s
  skl remove
  skl remove my-skill
  skl remove skill1 skill2 -y
  skl remove --global my-skill
  skl remove --all`, ansiBold, ansiReset),
		Args: cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if opts.All {
				opts.Skills = []string{"*"}
				opts.Agents = []string{"*"}
				opts.Yes = true
			}
			runRemove(args, opts)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&opts.Global, "global", "g", false, "Remove from global scope")
	f.StringArrayVarP(&opts.Agents, "agent", "a", nil, "Remove from specific agents (repeatable)")
	f.StringArrayVarP(&opts.Skills, "skill", "s", nil, "Skill names to remove (repeatable)")
	f.BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompts")
	f.BoolVar(&opts.All, "all", false, "Shorthand for --skill '*' --agent '*' -y")

	return cmd
}

// ─── list ──────────────────────────────────────────────────────────────────────

func buildListCmd() *cobra.Command {
	var globalFlag bool
	var projectFlag bool
	var agentFilter []string
	var jsonMode bool

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List installed skills",
		Aliases: []string{"ls"},
		Long: fmt.Sprintf(`List installed skills.

%sExamples:%s
  skl list
  skl ls -g
  skl list --json`, ansiBold, ansiReset),
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			var gFlag *bool
			if cmd.Flags().Changed("global") {
				t := true
				gFlag = &t
			} else if cmd.Flags().Changed("project") || projectFlag {
				f := false
				gFlag = &f
			}
			_ = globalFlag
			runListWithOpts(gFlag, agentFilter, jsonMode)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&globalFlag, "global", "g", false, "List global skills")
	f.BoolVarP(&projectFlag, "project", "p", false, "List project skills")
	f.StringArrayVarP(&agentFilter, "agent", "a", nil, "Filter by specific agents")
	f.BoolVar(&jsonMode, "json", false, "Output as JSON")

	return cmd
}

// ─── find ──────────────────────────────────────────────────────────────────────

func buildFindCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "find [query]",
		Short:   "Search the skills registry",
		Aliases: []string{"search", "f", "s"},
		Long: fmt.Sprintf(`Search the skills registry and install interactively.

%sExamples:%s
  skl find typescript
  skl search git`, ansiBold, ansiReset),
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			showLogo()
			fmt.Println()
			runFind(args)
		},
	}
}

// ─── update ────────────────────────────────────────────────────────────────────

func buildUpdateCmd() *cobra.Command {
	var opts UpdateOptions

	cmd := &cobra.Command{
		Use:     "update [skills...]",
		Short:   "Update installed skills",
		Aliases: []string{"check"},
		Long: fmt.Sprintf(`Update installed skills to their latest versions.

%sExamples:%s
  skl update
  skl update my-skill
  skl update -g`, ansiBold, ansiReset),
		Args: cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runUpdateWithOpts(args, opts)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&opts.Global, "global", "g", false, "Update global skills only")
	f.BoolVarP(&opts.Project, "project", "p", false, "Update project skills only")
	f.BoolVarP(&opts.Yes, "yes", "y", false, "Skip scope prompt")

	return cmd
}

// ─── upgrade ───────────────────────────────────────────────────────────────────

func buildUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade the skl CLI binary",
		Aliases: []string{"update-cli", "self-update"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runSelfUpdate(Version)
		},
	}
}

// ─── init ──────────────────────────────────────────────────────────────────────

func buildInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [name]",
		Short: "Scaffold a new skill",
		Long: fmt.Sprintf(`Initialize a new skill in the current directory.

Creates <name>/SKILL.md or ./SKILL.md if no name is given.

%sExamples:%s
  skl init
  skl init my-skill`, ansiBold, ansiReset),
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			showLogo()
			fmt.Println()
			runInitCmd(args)
		},
	}
}

// ─── experimental_install ──────────────────────────────────────────────────────

func buildInstallFromLockCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "experimental_install",
		Short: "Restore skills from skills-lock.json",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			showLogo()
			runInstallFromLock(yes)
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompts")
	return cmd
}

// ─── experimental_sync ─────────────────────────────────────────────────────────

func buildSyncCmd() *cobra.Command {
	var opts SyncOptions

	cmd := &cobra.Command{
		Use:   "experimental_sync",
		Short: "Sync skills from node_modules into agent directories",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			showLogo()
			runSync(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompts")
	return cmd
}

// ─── completion ────────────────────────────────────────────────────────────────

func buildCompletionCmd(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: fmt.Sprintf(`Generate a shell completion script for skl.

%sUsage:%s
  # Bash
  source <(skl completion bash)

  # Zsh
  skl completion zsh > "${fpath[1]}/_skl"

  # Fish
  skl completion fish | source`, ansiBold, ansiReset),
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = root.GenBashCompletion(os.Stdout)
			case "zsh":
				_ = root.GenZshCompletion(os.Stdout)
			case "fish":
				_ = root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				_ = root.GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
	return cmd
}

// ─── init command business logic ───────────────────────────────────────────────

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

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
