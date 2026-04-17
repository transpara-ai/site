package graph

import (
	"context"
	"embed"
	"log"
	"strings"
)

//go:embed personas/*.md
var personaFiles embed.FS

// personaCategory maps agent slug names to their UI category.
var personaCategory = map[string]string{
	// Care
	"steward":  "care",
	"witness":  "care",
	"mourner":  "care",
	"harmony":  "care",
	"teacher":  "care",
	// Governance
	"ceo":        "governance",
	"cto":        "governance",
	"pm":         "governance",
	"guardian":   "governance",
	"advocate":   "governance",
	"dissenter":  "governance",
	"philosopher": "governance",
	// Knowledge
	"historian": "knowledge",
	"librarian": "knowledge",
	"explorer":  "knowledge",
	"analyst":   "knowledge",
	"research":  "knowledge",
	// Product
	"scout":     "product",
	"architect": "product",
	"builder":   "product",
	"designer":  "product",
	"critic":    "product",
	"tester":    "product",
	"observer":  "product",
	// Outward
	"storyteller":      "outward",
	"newcomer":         "outward",
	"inhabitant":       "outward",
	"growth":           "outward",
	"customer-success": "outward",
	// Resource
	"simplifier":  "resource",
	"legal":       "resource",
	"philanthropy": "resource",
	"budget":      "resource",
	"finance":     "resource",
}

// personaActive is the set of personas shown to users on the Agents page.
// Pipeline/internal agents are excluded from the user-facing list.
var personaActive = map[string]bool{
	"steward": true, "witness": true, "mourner": true, "harmony": true, "teacher": true,
	"ceo": true, "cto": true, "pm": true, "guardian": true, "advocate": true,
	"dissenter": true, "philosopher": true,
	"historian": true, "librarian": true, "explorer": true, "analyst": true, "research": true,
	"scout": true, "architect": true, "builder": true, "designer": true,
	"critic": true, "tester": true, "observer": true,
	"storyteller": true, "newcomer": true, "inhabitant": true, "growth": true, "customer-success": true,
	"simplifier": true, "legal": true, "philanthropy": true, "budget": true, "finance": true,
}

// personaModel maps agent slugs to the preferred model (default: sonnet).
var personaModel = map[string]string{
	"dissenter":  "opus",
	"philosopher": "opus",
}

// statusPrefix is the leading HTML-comment marker declaring a persona's
// maturity on the source .md file, e.g. `<!-- Status: absorbed -->`.
const statusPrefix = "<!-- Status:"

// inactiveStatuses are persona maturities that disqualify the persona
// from being Active in the UI, regardless of the personaActive map.
// Source .md file is the authority: a role absorbed or retired in the
// hive repo must not appear on the site's user-facing agents page.
var inactiveStatuses = map[string]bool{
	"absorbed": true,
	"retired":  true,
}

// parsePersonaFile extracts (display, description, status) from an agent .md file.
// display: first `# Heading` line, stripped of `# `.
// description: first non-empty line that doesn't start with `#` or `<!--` or `>`.
// status: value of the leading `<!-- Status: X -->` comment, or "" if none.
func parsePersonaFile(content string) (display, description, status string) {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, statusPrefix) && status == "" {
			status = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, statusPrefix), "-->"))
			continue
		}
		if strings.HasPrefix(trimmed, "<!--") {
			continue // skip other metadata comments (Absorbed-By, Absorbs, etc.)
		}
		if strings.HasPrefix(trimmed, "# ") && display == "" {
			display = strings.TrimPrefix(trimmed, "# ")
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue // skip ## headings
		}
		if strings.HasPrefix(trimmed, ">") {
			continue // skip blockquotes (soul line)
		}
		if description == "" {
			description = trimmed
			break
		}
	}
	return
}

// SeedAgentPersonas reads embedded personas/*.md files and upserts them into
// the agent_personas table. Safe to call on every startup — idempotent.
func (s *Store) SeedAgentPersonas(ctx context.Context) {
	entries, err := personaFiles.ReadDir("personas")
	if err != nil {
		log.Printf("personas: read dir: %v", err)
		return
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := personaFiles.ReadFile("personas/" + entry.Name())
		if err != nil {
			log.Printf("personas: read %s: %v", entry.Name(), err)
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		display, description, status := parsePersonaFile(string(data))
		if display == "" {
			display = strings.Title(name) // fallback
		}
		if status == "" {
			status = "ready"
		}

		category := personaCategory[name]
		if category == "" {
			category = "general"
		}

		model := personaModel[name]
		if model == "" {
			model = "sonnet"
		}

		// Source .md file overrides personaActive: absorbed/retired personas
		// are forced inactive even if the hardcoded map says otherwise.
		active := personaActive[name]
		if inactiveStatuses[status] {
			active = false
		}

		if err := s.UpsertAgentPersona(ctx, AgentPersona{
			Name:        name,
			Display:     display,
			Description: description,
			Category:    category,
			Prompt:      string(data),
			Model:       model,
			Active:      active,
			Status:      status,
		}); err != nil {
			log.Printf("personas: upsert %s: %v", name, err)
			continue
		}
		count++
	}

	log.Printf("personas: seeded %d agent personas", count)
}
