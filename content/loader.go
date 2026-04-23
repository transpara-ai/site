// Package content loads embedded blog posts from markdown files.
package content

import (
	"bytes"
	"embed"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/transpara-ai/site/views"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

//go:embed posts/*.md
var postsFS embed.FS

var (
	monthYear = regexp.MustCompile(`·\s+(\w+)\s+(\d{4})`)
	postNum   = regexp.MustCompile(`^post(\d+)`)
)

// LoadPosts reads all embedded markdown posts and returns them in chronological order.
func LoadPosts() ([]views.Post, error) {
	entries, err := postsFS.ReadDir("posts")
	if err != nil {
		return nil, fmt.Errorf("read posts dir: %w", err)
	}

	md := goldmark.New(goldmark.WithExtensions(extension.Table))
	var posts []views.Post

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		raw, err := postsFS.ReadFile("posts/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}

		post, err := parsePost(md, e.Name(), raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		posts = append(posts, post)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Order < posts[j].Order
	})

	return posts, nil
}

func parsePost(md goldmark.Markdown, filename string, raw []byte) (views.Post, error) {
	lines := strings.SplitN(string(raw), "\n", -1)

	// Title: first line starting with "# "
	var title string
	for _, l := range lines {
		if strings.HasPrefix(l, "# ") {
			title = strings.TrimPrefix(l, "# ")
			break
		}
	}

	// Summary: the subtitle line (line after the title, before the author).
	// Formats vary: *italic*, _italic_, ## heading, plain text.
	summary := extractSummary(lines)

	// Date: parse "· Month Year" from byline
	date := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if m := monthYear.FindStringSubmatch(string(raw)); m != nil {
		year, _ := strconv.Atoi(m[2])
		month := parseMonth(m[1])
		day := 1
		// Use post number as day offset for ordering
		if nm := postNum.FindStringSubmatch(filename); nm != nil {
			n, _ := strconv.Atoi(nm[1])
			day = n
			if day > 28 {
				day = 28
			}
		}
		date = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}

	// Slug: strip "postNN-" prefix and ".md" suffix
	slug := strings.TrimSuffix(filename, ".md")
	if idx := strings.Index(slug, "-"); idx > 0 {
		slug = slug[idx+1:]
	}

	// Body: convert full markdown to HTML (skip the header metadata lines)
	bodyStart := findBodyStart(lines)
	bodyMD := strings.Join(lines[bodyStart:], "\n")

	var buf bytes.Buffer
	if err := md.Convert([]byte(bodyMD), &buf); err != nil {
		return views.Post{}, fmt.Errorf("convert markdown: %w", err)
	}

	// Post number from filename for stable ordering.
	order := 0
	if nm := postNum.FindStringSubmatch(filename); nm != nil {
		order, _ = strconv.Atoi(nm[1])
	}

	return views.Post{
		Slug:    slug,
		Title:   title,
		Summary: summary,
		Date:    date,
		Body:    buf.String(),
		Order:   order,
	}, nil
}

// findBodyStart skips past the title, subtitle, author, and --- separator(s).
// Only searches the first 15 lines for --- separators so that section dividers
// within the body content are not mistaken for header delimiters.
func findBodyStart(lines []string) int {
	limit := 15
	if limit > len(lines) {
		limit = len(lines)
	}
	firstHR := -1
	for i := 0; i < limit; i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			if firstHR >= 0 {
				// Second --- in header area — body starts after it.
				return i + 1
			}
			firstHR = i
		}
	}
	if firstHR >= 0 {
		// Single --- in header area — body starts after it.
		return firstHR + 1
	}
	// No separator found — start after title.
	for i, l := range lines {
		if strings.HasPrefix(l, "# ") {
			return i + 1
		}
	}
	return 0
}

// extractSummary finds the subtitle line between the title and the author/separator.
func extractSummary(lines []string) string {
	// Find the title line, then look at the lines between it and the author/separator.
	titleIdx := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "# ") {
			titleIdx = i
			break
		}
	}
	if titleIdx < 0 {
		return ""
	}

	// Scan lines after title, looking for the subtitle before we hit author or separator.
	for i := titleIdx + 1; i < len(lines) && i < titleIdx+6; i++ {
		l := strings.TrimSpace(lines[i])
		if l == "" {
			continue
		}
		// Stop at separators and author lines.
		if l == "---" {
			continue
		}
		if strings.Contains(l, "·") || strings.Contains(l, "|") && strings.Contains(l, "2026") {
			break
		}
		if strings.HasPrefix(l, "**Matt") || strings.HasPrefix(l, "Matt Searles") || strings.HasPrefix(l, "+Claude") {
			break
		}

		// Strip markdown formatting markers.
		s := l
		// Bold+italic: ***text***, ___text___
		for _, wrap := range []string{"***", "___", "**", "__", "*", "_"} {
			if strings.HasPrefix(s, wrap) && strings.HasSuffix(s, wrap) && len(s) > 2*len(wrap) {
				s = s[len(wrap) : len(s)-len(wrap)]
				break
			}
		}
		// Strip ## heading prefix
		s = strings.TrimPrefix(s, "## ")
		s = strings.TrimSpace(s)

		if len(s) > 200 {
			s = s[:200] + "..."
		}
		return s
	}
	return ""
}

func parseMonth(s string) time.Month {
	months := map[string]time.Month{
		"January": time.January, "February": time.February, "March": time.March,
		"April": time.April, "May": time.May, "June": time.June,
		"July": time.July, "August": time.August, "September": time.September,
		"October": time.October, "November": time.November, "December": time.December,
	}
	if m, ok := months[s]; ok {
		return m
	}
	return time.January
}
