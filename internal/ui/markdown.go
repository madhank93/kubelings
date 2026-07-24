package ui

import (
	"os"
	"strings"

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

// shellWidth is the terminal width to wrap the shell's task/hint/solution to,
// detected from stdout with a sane fallback for non-terminals.
func shellWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 20 {
		return w
	}
	return 100
}
