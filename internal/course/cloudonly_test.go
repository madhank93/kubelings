package course

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeCourse builds a throwaway repo root with one lesson that has tasks and
// one that does not, so the cloud-only wiring can be tested without depending
// on which real lessons happen to be in the registry.
func fakeCourse(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(dir, body string) {
		d := filepath.Join(root, "courses", "kubelings", "module-1", dir)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "index.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("1.with-tasks", "---\ntitle: With Tasks\ntasks:\n  verify_done:\n    run: true\n---\n")
	write("2.no-tasks", "---\ntitle: No Tasks\n---\n")
	return root
}

func lessonByName(t *testing.T, root, name string) Lesson {
	t.Helper()
	mods, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, m := range mods {
		for _, l := range m.Lessons {
			if l.Name == name {
				return l
			}
		}
	}
	t.Fatalf("lesson %q not found", name)
	return Lesson{}
}

func writeRegistry(t *testing.T, root, body string) {
	t.Helper()
	d := filepath.Join(root, ".labctl")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "cloud-only.tsv"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCloudOnlyMissingRegistryIsInert(t *testing.T) {
	root := fakeCourse(t)
	if l := lessonByName(t, root, "with-tasks"); l.CloudOnly {
		t.Error("no registry file, yet lesson is CloudOnly")
	}
	if got := LoadCloudOnly(root); len(got) != 0 {
		t.Errorf("missing file should load empty, got %v", got)
	}
}

func TestCloudOnlyRegistryRowFlipsLesson(t *testing.T) {
	root := fakeCourse(t)
	writeRegistry(t, root, "# comment\n\nwith-tasks\tit reboots the node\n")
	l := lessonByName(t, root, "with-tasks")
	if !l.CloudOnly {
		t.Fatal("registry row did not set CloudOnly")
	}
	if l.CloudOnlyReason != "it reboots the node" {
		t.Errorf("reason = %q", l.CloudOnlyReason)
	}
	if l.Type != "lab" {
		t.Errorf("cloud-only must not change Type, got %q", l.Type)
	}
}

func TestCloudOnlyIgnoresTaskLessLesson(t *testing.T) {
	root := fakeCourse(t)
	writeRegistry(t, root, "no-tasks\tit reboots the node\n")
	if l := lessonByName(t, root, "no-tasks"); l.CloudOnly {
		t.Error("a task-less lesson has nothing to run anywhere; it must stay a reading")
	}
}

func TestCloudOnlyFrontmatterFallback(t *testing.T) {
	root := fakeCourse(t)
	d := filepath.Join(root, "courses", "kubelings", "module-1", "1.with-tasks", "index.md")
	body := "---\ntitle: With Tasks\ncloudOnly: true\ntasks:\n  verify_done:\n    run: true\n---\n"
	if err := os.WriteFile(d, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if l := lessonByName(t, root, "with-tasks"); !l.CloudOnly {
		t.Error("cloudOnly frontmatter key did not set CloudOnly")
	}
}

func TestCourseURL(t *testing.T) {
	root := t.TempDir()
	if got, want := CourseURL(root), "https://labs.iximiuz.com/courses/"+defaultCourseSlug; got != want {
		t.Errorf("no slugs.tsv: got %q want %q", got, want)
	}
	d := filepath.Join(root, ".labctl")
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "slugs.tsv"), []byte("kubelings-course\tkubelings-abc123\n"), 0o644)
	if got, want := CourseURL(root), "https://labs.iximiuz.com/courses/kubelings-abc123"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
