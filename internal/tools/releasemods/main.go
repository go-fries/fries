// releasemods resolves the release manifest to local module directories.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/modfile"
)

type manifest struct {
	Sets map[string]struct {
		Version string   `yaml:"version"`
		Modules []string `yaml:"modules"`
	} `yaml:"module-sets"`
	Excluded []string `yaml:"excluded-modules"`
}

type releaseModule struct {
	dir     string
	version string
}

func main() {
	set := flag.String("modset", "", "module set from versions.yaml (empty selects all sets)")
	dir := flag.String("dir", "", "repository-relative module directory (empty selects all modules)")
	flag.Parse()
	log.SetFlags(0)
	if flag.NArg() != 0 {
		log.Fatal("releasemods accepts flags only")
	}
	modules, err := selectModules(".", *set, *dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, module := range modules {
		fmt.Printf("%s\t%s\n", module.dir, module.version)
	}
}

func selectModules(root, setName, dir string) ([]releaseModule, error) {
	data, err := os.ReadFile(filepath.Join(root, "versions.yaml"))
	if err != nil {
		return nil, err
	}
	var config manifest
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("versions.yaml: %w", err)
	}
	if setName != "" {
		if _, ok := config.Sets[setName]; !ok {
			return nil, fmt.Errorf("unknown module set %q", setName)
		}
	}
	versions := make(map[string]string)
	for name, set := range config.Sets {
		if setName != "" && name != setName {
			continue
		}
		if set.Version == "" {
			return nil, fmt.Errorf("module set %q has no version", name)
		}
		for _, path := range set.Modules {
			if _, ok := versions[path]; ok {
				return nil, fmt.Errorf("module %q is listed more than once", path)
			}
			if slices.Contains(config.Excluded, path) {
				return nil, fmt.Errorf("module %q is both released and excluded", path)
			}
			versions[path] = set.Version
		}
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("no release modules selected")
	}

	found := make(map[string]bool)
	var modules []releaseModule
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if slices.Contains([]string{".git", ".tools", ".coverage"}, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "go.mod" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := modfile.ParseLax(path, data, nil)
		if err != nil {
			return err
		}
		if file.Module == nil {
			return fmt.Errorf("%s has no module directive", path)
		}
		modulePath := file.Module.Mod.Path
		version, selected := versions[modulePath]
		if !selected {
			return nil
		}
		if found[modulePath] {
			return fmt.Errorf("module %q has multiple local directories", modulePath)
		}
		found[modulePath] = true
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		if strings.ContainsAny(rel, "\t\r\n") {
			return fmt.Errorf("module directory %q contains a tab or newline", rel)
		}
		if dir == "" || filepath.Clean(dir) == rel {
			modules = append(modules, releaseModule{dir: filepath.ToSlash(rel), version: version})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for path := range versions {
		if !found[path] {
			return nil, fmt.Errorf("release module %q has no local go.mod", path)
		}
	}
	if len(modules) == 0 {
		return nil, fmt.Errorf("directory %q is not in the selected release modules", dir)
	}
	slices.SortFunc(modules, func(a, b releaseModule) int { return strings.Compare(a.dir, b.dir) })
	return modules, nil
}
