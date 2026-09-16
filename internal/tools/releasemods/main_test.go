package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testManifest = `module-sets:
  stable:
    version: v4.3.0
    modules:
      - example.com/root/v4
      - example.com/root/alpha/v4
  experimental:
    version: v0.2.0
    modules:
      - example.com/root/beta
excluded-modules:
  - example.com/root/example/v4
  - example.com/root/tools
`

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "versions.yaml", testManifest)
	for dir, path := range map[string]string{
		".":                      "example.com/root/v4",
		"components/alpha":       "example.com/root/alpha/v4",
		"beta":                   "example.com/root/beta",
		"example":                "example.com/root/example/v4",
		"internal/tools":         "example.com/root/tools",
		".coverage/old-artifact": "example.com/root/v4",
	} {
		writeFile(t, root, filepath.Join(dir, "go.mod"), "module "+path+"\n\ngo 1.26.0\n")
	}
	return root
}

func TestSelectModules(t *testing.T) {
	t.Parallel()
	root := fixture(t)
	for _, tt := range []struct {
		name, set, dir string
		want           []releaseModule
	}{
		{"all", "", "", []releaseModule{{".", "v4.3.0"}, {"beta", "v0.2.0"}, {"components/alpha", "v4.3.0"}}},
		{"set", "stable", "", []releaseModule{{".", "v4.3.0"}, {"components/alpha", "v4.3.0"}}},
		{"root", "stable", ".", []releaseModule{{".", "v4.3.0"}}},
		{"nested directory", "stable", "./components/alpha/", []releaseModule{{"components/alpha", "v4.3.0"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := selectModules(root, tt.set, tt.dir)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("modules = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSelectionErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, set, dir, want string
	}{
		{"unknown set", "missing", "", "unknown module set"},
		{"excluded example", "", "example", "not in the selected release modules"},
		{"excluded tools", "", "internal/tools", "not in the selected release modules"},
		{"outside set", "stable", "beta", "not in the selected release modules"},
		{"outside repository", "", "../other", "not in the selected release modules"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := selectModules(fixture(t), tt.set, tt.dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestInvalidManifest(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, content, want string
	}{
		{"empty", "module-sets: {}", "no release modules"},
		{"malformed", "module-sets: [", "versions.yaml"},
		{"missing module", strings.ReplaceAll(testManifest, "example.com/root/alpha/v4", "example.com/missing"), "no local go.mod"},
		{"missing version", strings.ReplaceAll(testManifest, "version: v4.3.0", "version: ''"), "has no version"},
		{"duplicate", strings.ReplaceAll(testManifest, "example.com/root/alpha/v4", "example.com/root/v4"), "listed more than once"},
		{"excluded and released", testManifest + "  - example.com/root/v4\n", "both released and excluded"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := fixture(t)
			writeFile(t, root, "versions.yaml", tt.content)
			_, err := selectModules(root, "", "")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestDuplicateLocalModule(t *testing.T) {
	t.Parallel()
	root := fixture(t)
	writeFile(t, root, "duplicate/go.mod", "module example.com/root/v4\n")
	_, err := selectModules(root, "", "")
	if err == nil || !strings.Contains(err.Error(), "multiple local directories") {
		t.Fatalf("error = %v, want duplicate local module error", err)
	}
}
