// Package course discovers the kubelings course structure (modules and lessons)
// from courses/kubelings on disk. It is read-only — execution is the runner's job.
package course

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Lesson is one runnable (or content-only) scenario.
type Lesson struct {
	Module      string // module dir name, e.g. "module-2"
	ModuleTitle string
	Order       int    // numeric prefix of the lesson dir
	Name        string // dir basename minus "N.", e.g. "rolling-update"
	Title       string
	Description string
	Playground  string
	Dir         string // absolute lesson dir
	HasTasks    bool
	Hint        string
	Solution    string
}

// Module groups lessons.
type Module struct {
	Name    string
	Title   string
	Order   int
	Lessons []Lesson
}

type frontmatter struct {
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Name        string                 `yaml:"name"`
	Playground  struct{ Name string }  `yaml:"playground"`
	Tasks       map[string]interface{} `yaml:"tasks"`
}

var (
	moduleDirRe = regexp.MustCompile(`^module-(\d+)$`)
	lessonDirRe = regexp.MustCompile(`^(\d+)\.(.+)$`)
)

// CourseDir returns courses/kubelings under root.
func CourseDir(root string) string { return filepath.Join(root, "courses", "kubelings") }

// Discover scans the course and returns modules in order, each with ordered lessons.
func Discover(root string) ([]Module, error) {
	base := CourseDir(root)
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	var mods []Module
	for _, e := range entries {
		if !e.IsDir() || !moduleDirRe.MatchString(e.Name()) {
			continue
		}
		mo, _ := strconv.Atoi(moduleDirRe.FindStringSubmatch(e.Name())[1])
		m := Module{Name: e.Name(), Order: mo}
		if fm, err := readFrontmatter(filepath.Join(base, e.Name(), "0.index.md")); err == nil {
			m.Title = fm.Title
		}
		lessons, _ := os.ReadDir(filepath.Join(base, e.Name()))
		for _, le := range lessons {
			if !le.IsDir() {
				continue
			}
			mm := lessonDirRe.FindStringSubmatch(le.Name())
			if mm == nil {
				continue
			}
			ldir := filepath.Join(base, e.Name(), le.Name())
			idx := filepath.Join(ldir, "index.md")
			fm, err := readFrontmatter(idx)
			if err != nil {
				continue
			}
			order, _ := strconv.Atoi(mm[1])
			ls := Lesson{
				Module: e.Name(), ModuleTitle: m.Title, Order: order,
				Name: mm[2], Title: fm.Title, Description: strings.TrimSpace(fm.Description),
				Playground: fm.Playground.Name, Dir: ldir, HasTasks: len(fm.Tasks) > 0,
			}
			ls.Hint = extractDetails(filepath.Join(ldir, "unit-1.md"), "hint")
			ls.Solution = extractDetails(filepath.Join(ldir, "unit-1.md"), "solution")
			m.Lessons = append(m.Lessons, ls)
		}
		sort.Slice(m.Lessons, func(i, j int) bool { return m.Lessons[i].Order < m.Lessons[j].Order })
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].Order < mods[j].Order })
	return mods, nil
}

// readFrontmatter parses the YAML block between the first two "---" lines.
func readFrontmatter(path string) (frontmatter, error) {
	var fm frontmatter
	b, err := os.ReadFile(path)
	if err != nil {
		return fm, err
	}
	lines := strings.Split(string(b), "\n")
	var start, end = -1, -1
	for i, l := range lines {
		if strings.TrimRight(l, " \t") == "---" {
			if start == -1 {
				start = i
			} else {
				end = i
				break
			}
		}
	}
	if start == -1 || end == -1 {
		return fm, yaml.Unmarshal(b, &fm) // best effort
	}
	block := strings.Join(lines[start+1:end], "\n")
	return fm, yaml.Unmarshal([]byte(block), &fm)
}

// extractDetails returns the markdown body of a <details> whose <summary> contains
// the given keyword (case-insensitive), with the summary/tags stripped.
func extractDetails(path, keyword string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := string(b)
	low := strings.ToLower(text)
	kw := strings.ToLower(keyword)
	for searchFrom := 0; ; {
		open := strings.Index(low[searchFrom:], "<details>")
		if open == -1 {
			return ""
		}
		open += searchFrom
		close := strings.Index(low[open:], "</details>")
		if close == -1 {
			return ""
		}
		close += open
		block := text[open:close]
		if strings.Contains(strings.ToLower(block), kw) {
			// drop <details>, <summary>...</summary>
			body := block
			if i := strings.Index(strings.ToLower(body), "</summary>"); i != -1 {
				body = body[i+len("</summary>"):]
			} else {
				body = strings.TrimPrefix(body, text[open:open+len("<details>")])
			}
			return strings.TrimSpace(body)
		}
		searchFrom = close + len("</details>")
	}
}
