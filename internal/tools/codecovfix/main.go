package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const modulesPrefix = "github.com/go-fries/fries/"

var excludedDirs = []string{
	"internal",
}

var moduleRegex = regexp.MustCompile(`^github\.com/go-fries/fries/(.+)/v3$`)

func main() {
	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var modules []string
	err = filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		// skip excluded directories
		for _, dir := range excludedDirs {
			if strings.Contains(path, dir) {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() && filepath.Base(path) == "go.mod" {
			// read go.mod and get the module name
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Extract module name from go.mod
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "module ") {
					moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
					if strings.HasPrefix(moduleName, modulesPrefix) {
						modules = append(modules, moduleName)
					}
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Print the "fixes" content of the codecov configuration for each module
	for _, module := range modules {
		matches := moduleRegex.FindStringSubmatch(module)
		if len(matches) == 2 {
			moduleName := matches[1]
			fixesContent := fmt.Sprintf("%s::%s", module, moduleName+"/")
			fmt.Println(fixesContent)
		}
	}
}
