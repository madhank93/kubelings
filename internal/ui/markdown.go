package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// mdStyle is glamour's dark style with the literal heading prefixes ("# ",
// "## ", …) stripped — those are exactly the raw markers the render is meant to
// remove. Headings keep their color/weight, just without the ## text.
var mdStyle = func() ansi.StyleConfig {
	s := styles.DarkStyleConfig
	s.H1.Prefix, s.H2.Prefix, s.H3.Prefix = "", "", ""
	s.H4.Prefix, s.H5.Prefix, s.H6.Prefix = "", "", ""
	return s
}()

// renderMarkdown turns lesson markdown (task / hint / solution prose) into
// styled ANSI for the right-hand viewport and the cluster shell, so headings,
// bold, lists, and fenced code blocks render instead of showing their raw
// `##` / `**` / ``` source. On any error it returns the text unchanged — a
// plain but readable fallback.
func renderMarkdown(md string, width int) string {
	if strings.TrimSpace(md) == "" {
		return md
	}
	if width < 20 {
		width = 80
	}
	// Force the dark style + an explicit color profile so the output is
	// deterministic — glamour's auto-style falls back to a raw "notty" render
	// (leaving ## and ** literal) whenever stdout isn't a TTY, which is the case
	// when we render into a viewport string or a file the shell later cats.
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(mdStyle),
		glamour.WithColorProfile(termenv.ANSI256),
		glamour.WithWordWrap(width), // wrap to the pane / terminal width
		glamour.WithEmoji(),         // the lessons use ✅ / ☁ etc.
	)
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	// glamour pads with blank lines; trim so it sits flush in the viewport.
	return strings.Trim(out, "\n")
}

// d2Fence matches a generated diagram: ASCII art between the marker comments
// scripts/gen-diagrams.py writes. Same block the docs build swaps for an SVG.
// RE2 has no backreferences, so the closing marker's name is matched loosely;
// the generator is what guarantees the pair agrees.
var d2Fence = regexp.MustCompile("(?ms)^<!-- d2:([a-z0-9-]+) -->\n```text\n(.*?)\n```\n<!-- /d2:[a-z0-9-]+ -->\n?")

// diagramStyle keeps box art visually distinct from prose and from code the
// learner is meant to type.
var diagramStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))

// renderLesson renders lesson markdown, drawing embedded diagrams centered in
// the pane instead of handing them to glamour. Glamour would render the art as
// a code block: left-aligned, background-filled, and indented by its own
// margin — which reads as a command to run rather than a picture.
func renderLesson(md string, width int) string {
	locs := d2Fence.FindAllStringSubmatchIndex(md, -1)
	if len(locs) == 0 {
		return renderMarkdown(md, width)
	}
	var b strings.Builder
	prev := 0
	for _, loc := range locs {
		if prose := strings.TrimSpace(md[prev:loc[0]]); prose != "" {
			b.WriteString(renderMarkdown(prose, width) + "\n\n")
		}
		art := strings.TrimRight(md[loc[4]:loc[5]], "\n")
		b.WriteString(lipgloss.PlaceHorizontal(width, lipgloss.Center, diagramStyle.Render(art)) + "\n\n")
		prev = loc[1]
	}
	if rest := strings.TrimSpace(md[prev:]); rest != "" {
		b.WriteString(renderMarkdown(rest, width))
	}
	return strings.Trim(b.String(), "\n")
}

// md renders markdown through renderMarkdown, memoised per lesson/section/width.
// The cache is dropped whenever the course is re-read from disk (reload).
func (m *model) md(key, src string, width int) string {
	if strings.TrimSpace(src) == "" {
		return ""
	}
	ck := fmt.Sprintf("%s|%d", key, width)
	if out, ok := m.mdCache[ck]; ok {
		return out
	}
	out := renderLesson(src, width)
	if m.mdCache == nil {
		m.mdCache = map[string]string{}
	}
	m.mdCache[ck] = out
	return out
}

// shellWidth is the terminal width to wrap the shell's task/hint/solution to,
// detected from stdout with a sane fallback for non-terminals.
func shellWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 20 {
		return w
	}
	return 100
}
