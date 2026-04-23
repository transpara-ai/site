package content

import (
	"bytes"
	"strings"

	"github.com/transpara-ai/site/views"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var refMD = goldmark.New(goldmark.WithExtensions(extension.Table))

func parseGrammarPage(filename string, raw []byte) (views.RefPage, error) {
	lines := strings.SplitN(string(raw), "\n", -1)

	var title string
	for _, l := range lines {
		if strings.HasPrefix(l, "# ") {
			title = strings.TrimPrefix(l, "# ")
			break
		}
	}

	var summary string
	pastTitle := false
	for _, l := range lines {
		if strings.HasPrefix(l, "# ") {
			pastTitle = true
			continue
		}
		if !pastTitle {
			continue
		}
		l = strings.TrimSpace(l)
		if l == "" || l == "---" || strings.HasPrefix(l, "#") {
			continue
		}
		summary = l
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		break
	}

	order := 99
	slug := strings.TrimSuffix(filename, ".md")
	if len(slug) > 2 && slug[2] == '-' {
		n := 0
		for _, c := range slug[:2] {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		order = n
		slug = slug[3:]
	}

	bodyStart := 0
	for i, l := range lines {
		if strings.HasPrefix(l, "# ") {
			bodyStart = i + 1
			break
		}
	}
	bodyMD := strings.Join(lines[bodyStart:], "\n")

	var buf bytes.Buffer
	if err := refMD.Convert([]byte(bodyMD), &buf); err != nil {
		return views.RefPage{}, err
	}

	return views.RefPage{
		Slug:    slug,
		Title:   title,
		Summary: summary,
		Order:   order,
		Body:    buf.String(),
	}, nil
}
