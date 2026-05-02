package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sethcarney/mdm/internal/registry"
	"github.com/sethcarney/mdm/internal/ui"
)

const curatedAPIURL = "https://skills.sh/api/v1/skills/curated"

// ── API types ──────────────────────────────────────────────

type curatedSkill struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Installs   int    `json:"installs"`
	SourceType string `json:"sourceType"`
	InstallURL string `json:"installUrl"`
	URL        string `json:"url"`
}

type curatedOwner struct {
	Owner         string         `json:"owner"`
	TotalInstalls int            `json:"totalInstalls"`
	FeaturedRepo  string         `json:"featuredRepo"`
	FeaturedSkill string         `json:"featuredSkill"`
	Skills        []curatedSkill `json:"skills"`
}

type curatedResponse struct {
	Data          []curatedOwner `json:"data"`
	TotalOwners   int            `json:"totalOwners"`
	TotalSkills   int            `json:"totalSkills"`
}

// ── Command ────────────────────────────────────────────────

func buildOfficialCmd() *cobra.Command {
	var globalFlag bool
	var projectFlag bool

	cmd := &cobra.Command{
		Use:     "official",
		Short:   "Browse and install official curated skills",
		Aliases: []string{"curated"},
		Long: fmt.Sprintf(`Browse official skills from the skills.sh curated registry.

These are skills published by the companies that build the tools they teach —
first-party skills from the makers themselves.

%sExamples:%s
  mdm skills official
  mdm skills official -g`, ansiBold, ansiReset),
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			opts := AddOptions{
				Global:  globalFlag,
				Project: projectFlag,
			}
			runOfficial(opts)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&globalFlag, "global", "g", false, "Install globally")
	f.BoolVarP(&projectFlag, "project", "p", false, "Install for this project only")

	return cmd
}

// ── Run ────────────────────────────────────────────────────

func runOfficial(opts AddOptions) {
	fmt.Println()

	spin := ui.NewSpinner("Fetching official skills...")
	resp, err := fetchCuratedSkills()
	spin.Stop("")

	if err != nil || resp == nil || len(resp.Data) == 0 {
		fmt.Fprintf(os.Stderr, "%sCould not fetch official skills.%s\n", ansiText, ansiReset)
		return
	}

	// Header
	fmt.Printf("%sOfficial skills%s  %s%d publishers · %d skills%s\n\n",
		ansiBold, ansiReset,
		ansiDim, resp.TotalOwners, resp.TotalSkills, ansiReset)

	// Overview grouped by owner
	for _, owner := range resp.Data {
		names := make([]string, 0, len(owner.Skills))
		for _, s := range owner.Skills {
			names = append(names, s.Name)
		}
		preview := strings.Join(names, ", ")
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}
		fmt.Printf("  %s%-20s%s %s%s installs%s  %s%s%s\n",
			ansiText, owner.Owner, ansiReset,
			ansiDim, fmtInstalls(owner.TotalInstalls), ansiReset,
			ansiDim, preview, ansiReset)
	}
	fmt.Println()

	// Flatten all skills for multi-select
	var allSkills []curatedSkill
	for _, owner := range resp.Data {
		allSkills = append(allSkills, owner.Skills...)
	}

	options := make([]ui.UIOption, len(allSkills))
	for i, s := range allSkills {
		owner := ownerFromSource(s.Source)
		hint := fmt.Sprintf("%s  ·  %s installs", owner, fmtInstalls(s.Installs))
		options[i] = ui.UIOption{Label: s.Name, Value: s.ID, Hint: hint}
	}

	indices, ok := ui.UiMultiselect("Select skills to install", options, false, nil, nil)
	if !ok || len(indices) == 0 {
		fmt.Println("Cancelled.")
		return
	}

	// Group selected skills by installUrl (one runAdd call per repo)
	type group struct {
		installURL string
		slugs      []string
	}
	groupMap := map[string]*group{}
	var groupOrder []string
	for _, i := range indices {
		s := allSkills[i]
		if _, exists := groupMap[s.InstallURL]; !exists {
			groupMap[s.InstallURL] = &group{installURL: s.InstallURL}
			groupOrder = append(groupOrder, s.InstallURL)
		}
		groupMap[s.InstallURL].slugs = append(groupMap[s.InstallURL].slugs, s.Slug)
	}

	for _, url := range groupOrder {
		g := groupMap[url]
		addOpts := opts
		addOpts.Skills = g.slugs
		fmt.Println()
		runAdd(url, addOpts)
	}
}

// ── API fetch ──────────────────────────────────────────────

func fetchCuratedSkills() (*curatedResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body, status, err := registry.HttpGetText(ctx, curatedAPIURL)
	if err != nil || status != 200 {
		return nil, fmt.Errorf("status %d", status)
	}

	var resp curatedResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── Helpers ────────────────────────────────────────────────

func fmtInstalls(n int) string {
	s := fmt.Sprintf("%d", n)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}

func ownerFromSource(source string) string {
	if i := strings.Index(source, "/"); i != -1 {
		return source[:i]
	}
	return source
}
