package content

import (
	"bytes"
	"embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lovyou-ai/site/views"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

//go:embed reference/agent-primitives.md
var agentPrimitivesRaw []byte

//go:embed reference/product-layers.md
var productLayersRaw []byte

//go:embed reference/layers/*.md
var layerDocsFS embed.FS

//go:embed reference/grammars/*.md
var grammarsFS embed.FS

//go:embed reference/fundamentals/layer-*.md
var fundamentalsFS embed.FS

//go:embed reference/grammar.md
var baseGrammarRaw []byte

//go:embed reference/cognitive-grammar.md
var cognitiveGrammarRaw []byte

var (
	tableRow = regexp.MustCompile(`^\| \*\*(.+?)\*\* \| (.+) \|$`)
	primMD   = goldmark.New(goldmark.WithExtensions(extension.Table))
)

// LoadLayers parses all layer data:
// - Primitives from mind-zero layer files (layers 1-13)
// - Layer 0 primitives from 00-foundation.md (the given foundations)
// - Descriptions from product-layers.md and per-layer derivation docs
func LoadLayers() []views.Layer {
	// 1. Parse primitives from mind-zero files (layers 0-13).
	layers := parseFundamentals()

	// 2. For Layer 0, enrich with detailed specs from 00-foundation.md.
	layer0Specs := parseFoundationFile()
	if layer0Specs != nil {
		for i := range layers {
			if layers[i].Number == 0 {
				layers[i].Primitives = layer0Specs.Primitives
				break
			}
		}
	}

	// 3. Enrich layer descriptions from product-layers.md.
	productDescs := parseProductLayers()
	for i := range layers {
		if desc, ok := productDescs[layers[i].Number]; ok {
			if layers[i].Description != "" {
				layers[i].Description += "\n" + desc
			} else {
				layers[i].Description = desc
			}
		}
	}

	return layers
}

// parseFundamentals parses the mind-zero layer files to extract
// fundamental (ontological) primitives derived from layer gaps.
func parseFundamentals() []views.Layer {
	filenames := []struct {
		num  int
		name string
		file string
	}{
		{0, "Foundation", "layer-0-foundation.md"},
		{1, "Agency", "layer-1-agency.md"},
		{2, "Exchange", "layer-2-exchange.md"},
		{3, "Society", "layer-3-society.md"},
		{4, "Legal", "layer-4-legal.md"},
		{5, "Technology", "layer-5-technology.md"},
		{6, "Information", "layer-6-information.md"},
		{7, "Ethics", "layer-7-ethics.md"},
		{8, "Identity", "layer-8-identity.md"},
		{9, "Relationship", "layer-9-relationship.md"},
		{10, "Community", "layer-10-community.md"},
		{11, "Culture", "layer-11-culture.md"},
		{12, "Emergence", "layer-12-emergence.md"},
		{13, "Existence", "layer-13-existence.md"},
	}

	fundGroup := regexp.MustCompile(`^### Group [A-Z] — (.+?)(?:\s*\(.*\))?$`)
	fundRow := regexp.MustCompile(`^\| \*\*(.+?)\*\* \| (.+?) \| (.+?) \|$`)
	gapRe := regexp.MustCompile(`(?i)gap`)
	transitionRe := regexp.MustCompile(`(?i)transition`)

	var layers []views.Layer
	for _, f := range filenames {
		raw, err := fundamentalsFS.ReadFile("reference/fundamentals/" + f.file)
		if err != nil {
			continue
		}

		layer := views.Layer{
			Number: f.num,
			Name:   f.name,
		}

		lines := strings.Split(string(raw), "\n")
		var curGroup string
		var derivLines []string
		pastTitle := false
		inPrimitives := false

		for _, line := range lines {
			line = strings.TrimRight(line, "\r")

			// Extract layer name from title.
			if strings.HasPrefix(line, "# Layer") && !pastTitle {
				pastTitle = true
				// e.g. "# Layer 1 — Agency"
				if idx := strings.Index(line, "— "); idx >= 0 {
					layer.Name = strings.TrimSpace(line[idx+len("— "):])
				} else if idx := strings.Index(line, "- "); idx >= 0 {
					layer.Name = strings.TrimSpace(line[idx+2:])
				}
				continue
			}

			// Detect primitives section.
			if strings.HasPrefix(line, "## Primitives") {
				inPrimitives = true
				continue
			}

			// Detect end of derivation / start of other sections.
			if strings.HasPrefix(line, "## ") && pastTitle && !inPrimitives {
				heading := strings.ToLower(line)
				if gapRe.MatchString(heading) {
					// Try to extract gap text from following lines.
				} else if transitionRe.MatchString(heading) {
					// Try to extract transition.
				}
				// Collect derivation from the Derivation section.
				if strings.HasPrefix(line, "## Derivation") {
					derivLines = nil // reset, we'll collect lines after this
					continue
				}
				if len(derivLines) > 0 {
					// We hit a new section, flush derivation.
					md := strings.TrimSpace(strings.Join(derivLines, "\n"))
					var buf bytes.Buffer
					primMD.Convert([]byte(md), &buf)
					layer.Description = buf.String()
					derivLines = nil
				}
				continue
			}

			// Collect derivation lines.
			if pastTitle && !inPrimitives && derivLines != nil {
				derivLines = append(derivLines, line)
			}

			// Parse group headings.
			if m := fundGroup.FindStringSubmatch(line); m != nil {
				curGroup = m[1]
				continue
			}

			// Parse primitive rows (3-column: Name | Definition | Derivation).
			if inPrimitives {
				if m := fundRow.FindStringSubmatch(line); m != nil {
					layer.Primitives = append(layer.Primitives, views.Primitive{
						Name:       m[1],
						Slug:       slugify(m[1]),
						Layer:      f.num,
						LayerName:  layer.Name,
						Group:      curGroup,
						Description: strings.TrimSpace(m[2]),
						Derivation: strings.TrimSpace(m[3]),
					})
				}
			}
		}

		// Flush any remaining derivation.
		if len(derivLines) > 0 {
			md := strings.TrimSpace(strings.Join(derivLines, "\n"))
			var buf bytes.Buffer
			primMD.Convert([]byte(md), &buf)
			layer.Description = buf.String()
		}

		// Extract gap and transition from the derivation text.
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimRight(line, "\r")
			if strings.HasPrefix(line, "**Gap from Layer") {
				if idx := strings.Index(line, ":**"); idx >= 0 {
					layer.Gap = strings.TrimSpace(strings.TrimSuffix(line[idx+3:], "**"))
				}
			}
			if strings.HasPrefix(line, "**Transition:") {
				layer.Transition = strings.TrimSpace(strings.TrimPrefix(line, "**Transition:**"))
				layer.Transition = strings.TrimSpace(strings.TrimSuffix(layer.Transition, "**"))
			}
		}

		// For mind-zero files, extract gap/transition from "Known Gap" and key transition line.
		if layer.Gap == "" {
			for _, line := range strings.Split(string(raw), "\n") {
				line = strings.TrimRight(line, "\r")
				if strings.Contains(line, "key transition:") || strings.Contains(line, "Core Transition") {
					// e.g. "### The key transition: observer to participant"
					if idx := strings.Index(strings.ToLower(line), "transition:"); idx >= 0 {
						layer.Transition = strings.TrimSpace(line[idx+len("transition:"):])
					}
				}
			}
		}

		layers = append(layers, layer)
	}

	return layers
}

// parseFoundationFile parses 00-foundation.md for Layer 0 primitives with full specs.
func parseFoundationFile() *views.Layer {
	raw, err := layerDocsFS.ReadFile("reference/layers/00-foundation.md")
	if err != nil {
		return nil
	}

	layer := &views.Layer{
		Number:     0,
		Name:       "Foundation",
		Gap:        "None — this is the base layer.",
		Transition: "Nothing → Something",
	}

	lines := strings.Split(string(raw), "\n")
	var curGroup string
	var curPrim *views.Primitive
	var bodyLines []string

	// Foundation uses ### for primitives (not ####).
	foundationPrimHeading := regexp.MustCompile(`^### (.+)`)
	foundationGroupHeading := regexp.MustCompile(`^## Group \d+ — (.+)`)

	flushPrim := func() {
		if curPrim != nil {
			if len(bodyLines) > 0 {
				md := strings.TrimSpace(strings.Join(bodyLines, "\n"))
				var buf bytes.Buffer
				primMD.Convert([]byte(md), &buf)
				curPrim.Notes = buf.String()
			}
			layer.Primitives = append(layer.Primitives, *curPrim)
			curPrim = nil
		}
		bodyLines = nil
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if m := foundationGroupHeading.FindStringSubmatch(line); m != nil {
			flushPrim()
			curGroup = m[1]
			continue
		}

		if m := foundationPrimHeading.FindStringSubmatch(line); m != nil {
			flushPrim()
			name := m[1]
			curPrim = &views.Primitive{
				Name:      name,
				Slug:      slugify(name),
				Layer:     0,
				LayerName: "Foundation",
				Group:     curGroup,
			}
			continue
		}

		if curPrim != nil {
			if m := tableRow.FindStringSubmatch(line); m != nil {
				key := m[1]
				val := strings.TrimSpace(m[2])
				switch key {
				case "Subscribes to":
					curPrim.SubscribesTo = val
				case "Emits":
					curPrim.Emits = val
				case "Depends on":
					curPrim.DependsOn = val
				case "State":
					curPrim.State = val
				case "Intelligent", "Mechanical", "Both":
					curPrim.Intelligent = key + ": " + val
				}
				continue
			}
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || trimmed == "| | |" || trimmed == "|---|---|" {
				continue
			}
			if !strings.HasPrefix(trimmed, "##") && !strings.HasPrefix(trimmed, "---") {
				if curPrim.Description == "" {
					curPrim.Description = trimmed
				} else {
					bodyLines = append(bodyLines, trimmed)
				}
			}
		}
	}
	flushPrim()

	if len(layer.Primitives) == 0 {
		return nil
	}
	return layer
}

// parseProductLayers extracts per-layer product descriptions from product-layers.md.
// Returns map[layerNumber] → rendered HTML description.
func parseProductLayers() map[int]string {
	lines := strings.Split(string(productLayersRaw), "\n")
	result := make(map[int]string)

	plHeading := regexp.MustCompile(`^### Layer (\d+):`)
	var curNum int = -1
	var curLines []string

	flush := func() {
		if curNum >= 0 && len(curLines) > 0 {
			md := strings.Join(curLines, "\n")
			var buf bytes.Buffer
			primMD.Convert([]byte(md), &buf)
			result[curNum] = buf.String()
		}
		curLines = nil
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if m := plHeading.FindStringSubmatch(line); m != nil {
			flush()
			curNum, _ = strconv.Atoi(m[1])
			continue
		}
		if curNum >= 0 {
			// Stop at next ### or --- separator.
			if strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "## ") {
				flush()
				curNum = -1
				continue
			}
			if strings.TrimSpace(line) == "---" {
				flush()
				curNum = -1
				continue
			}
			curLines = append(curLines, line)
		}
	}
	flush()
	return result
}

// LoadAgentPrimitives parses agent-primitives.md into primitives.
func LoadAgentPrimitives() []views.Primitive {
	lines := strings.Split(string(agentPrimitivesRaw), "\n")

	var prims []views.Primitive
	var category string

	catHeading := regexp.MustCompile(`^### (Structural|Operational|Relational|Modal) \((\d+)\)`)
	agentRow := regexp.MustCompile(`^\| \*\*(.+?)\*\* \| (.+?) \| (.+?) \| (.+?) \| (.+?) \| (.+?) \| (.+?) \|$`)

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		if m := catHeading.FindStringSubmatch(line); m != nil {
			category = m[1]
			continue
		}

		if m := agentRow.FindStringSubmatch(line); m != nil {
			prims = append(prims, views.Primitive{
				Name:         m[1],
				Slug:         "agent-" + slugify(m[1]),
				Layer:        -1,
				LayerName:    "Agent",
				Group:        category,
				Description:  m[2],
				SubscribesTo: m[3],
				Emits:        m[4],
				DependsOn:    m[5],
				State:        m[6],
				Intelligent:  m[7],
			})
		}
	}

	return prims
}

func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// LoadGrammars reads composition grammar markdown files.
func LoadGrammars() ([]views.RefPage, error) {
	entries, err := grammarsFS.ReadDir("reference/grammars")
	if err != nil {
		return nil, fmt.Errorf("read grammars dir: %w", err)
	}

	var pages []views.RefPage
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		raw, err := grammarsFS.ReadFile("reference/grammars/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}

		page, err := parseGrammarPage(e.Name(), raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		pages = append(pages, page)
	}

	return pages, nil
}

// LoadBaseGrammar renders the base graph grammar markdown to HTML.
func LoadBaseGrammar() string {
	var buf bytes.Buffer
	primMD.Convert(baseGrammarRaw, &buf)
	return buf.String()
}

// LoadCognitiveGrammar renders the cognitive grammar markdown to HTML.
func LoadCognitiveGrammar() string {
	var buf bytes.Buffer
	primMD.Convert(cognitiveGrammarRaw, &buf)
	return buf.String()
}
