package course

import (
	"os"
	"path/filepath"
	"strings"
)

// defaultCourseSlug is the published course, used when .labctl/slugs.tsv is
// missing or does not carry a `kubelings-course` row.
const defaultCourseSlug = "kubelings-dbd840c8"

// LoadCloudOnly reads .labctl/cloud-only.tsv and returns lesson name -> reason.
//
// Cloud-only lessons need real-VM/host access (systemctl, sysctl, static pod
// manifests, etcd on disk, a node reboot). The local runner deliberately
// confines lesson scripts to the kind node container, so those lessons can only
// be offered on iximiuz Labs. The registry lives here rather than in lesson
// frontmatter because `labctl content push` rejects unknown frontmatter keys.
//
// A missing file yields an empty map — i.e. exactly today's behavior.
func LoadCloudOnly(root string) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(filepath.Join(root, ".labctl", "cloud-only.tsv"))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		name, reason, _ := strings.Cut(line, "\t")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = strings.TrimSpace(reason)
	}
	return out
}

// CourseURL is the published course page on iximiuz Labs, resolved from
// .labctl/slugs.tsv so a re-published course doesn't leave a stale link behind.
func CourseURL(root string) string {
	slug := defaultCourseSlug
	if b, err := os.ReadFile(filepath.Join(root, ".labctl", "slugs.tsv")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			k, v, ok := strings.Cut(strings.TrimRight(line, "\r"), "\t")
			if ok && strings.TrimSpace(k) == "kubelings-course" {
				if v = strings.TrimSpace(v); v != "" {
					slug = v
				}
				break
			}
		}
	}
	return "https://labs.iximiuz.com/courses/" + slug
}
