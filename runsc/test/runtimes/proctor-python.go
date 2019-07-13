// Copyright 2019 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Binary proctor is a utility that facilitates language testing for Pyhton373.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

var (
	list      = flag.Bool("list", false, "list all available tests")
	test      = flag.String("test", "", "run a single test from the list of available tests")
	version   = flag.Bool("v", false, "print out the version of node that is installed")
	dir       = "/python-cpython-0b8794d"
	testRegEx = regexp.MustCompile(`^test_.+\.py$`)
)

func main() {
	flag.Parse()

	if *list && *test != "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *list {
		listTests()
		return
	}
	if *version {
		fmt.Println("Python version: 3.7.3 is installed.")
		return
	}
	runTest(*test)
}

func listTests() {
	var files []string
	root := filepath.Join(dir, "Lib/test")

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		name := filepath.Base(path)

		if info.IsDir() || !testRegEx.MatchString(name) {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to walk %q: %v", root, err)
	}

	for _, file := range files {
		fmt.Println(file)
	}
}

func runTest(test string) {
	args := []string{"-m", "test"}
	if test != "" {
		args = append(args, test)
	}
	cmd := exec.Command("./python", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run: %v", err)
	}
}
