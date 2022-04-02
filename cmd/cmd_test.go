/*
 * Copyright 2021-2022 JetBrains s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

// Provides simple CLI tests for all supported platforms.

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/JetBrains/qodana-cli/core"
)

var testOptions = &core.QodanaOptions{
	ResultsDir:            "./results",
	CacheDir:              "./cache",
	ProjectDir:            "./project",
	Linter:                "jetbrains/qodana-jvm-community:2021.3",
	SourceDirectory:       "./src",
	DisableSanity:         true,
	RunPromo:              "true",
	Baseline:              "qodana.sarif.json",
	BaselineIncludeAbsent: true,
	SaveReport:            true,
	ShowReport:            true,
	Port:                  8888,
	Property:              "foo=bar",
	Script:                "default",
	FailThreshold:         "0",
	Changes:               true,
	SendReport:            true,
	AnalysisId:            "id",
	Env:                   []string{"A=B"},
	Volumes:               []string{"/tmp/foo:/tmp/foo"},
	User:                  "1001:1001",
	PrintProblems:         true,
	ProfileName:           "Default",
}

func isGitHubAction() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// TestVersion verifies that the version command returns the correct version
func TestVersion(t *testing.T) {
	b := bytes.NewBufferString("")
	command := NewRootCommand()
	command.SetOut(b)
	command.SetArgs([]string{"-v"})
	err := command.Execute()
	if err != nil {
		t.Fatal(err)
	}
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	expected := fmt.Sprintf("qodana version %s\n", core.Version)
	actual := string(out)
	if expected != actual {
		t.Fatalf("expected \"%s\" got \"%s\"", expected, actual)
	}
}

// TestHelp verifies that the help text is returned when running the tool with the flag or without it.
func TestHelp(t *testing.T) {
	out := bytes.NewBufferString("")
	command := NewRootCommand()
	command.SetOut(out)
	command.SetArgs([]string{"-h"})
	err := command.Execute()
	if err != nil {
		t.Fatal(err)
	}
	output, err := ioutil.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}
	expected := string(output)

	out = bytes.NewBufferString("")
	command = NewRootCommand()
	command.SetOut(out)
	command.SetArgs([]string{})
	err = command.Execute()
	if err != nil {
		t.Fatal(err)
	}
	output, err = ioutil.ReadAll(out)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(output)

	if expected != actual {
		t.Fatalf("expected \"%s\" got \"%s\"", expected, actual)
	}
}

func TestInitCommand(t *testing.T) {
	projectPath := "/tmp/qodana_init"
	err := os.MkdirAll(projectPath, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("/tmp/qodana_init/hello.py", []byte("print(\"Hello\")"), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	out := bytes.NewBufferString("")
	command := NewInitCommand()
	command.SetOut(out)
	command.SetArgs([]string{"-i", projectPath})
	err = command.Execute()
	if err != nil {
		t.Fatal(err)
	}

	qodanaYaml := core.LoadQodanaYaml(projectPath)

	if qodanaYaml.Linter != core.QDPY {
		t.Fatalf("expected \"%s\", but got %s", core.QDPY, qodanaYaml.Linter)
	}

	err = os.RemoveAll(projectPath)
	if err != nil {
		t.Fatal(err)
	}
}

// TestScanFlags verify that the option struct is converted to the wanted Qodana Docker options.
func TestScanFlags(t *testing.T) {
	expected := strings.Join([]string{
		"--save-report",
		"--source-directory",
		"./src",
		"--disable-sanity",
		"--profile-name",
		"Default",
		"--run-promo true",
		"--baseline",
		"qodana.sarif.json",
		"--baseline-include-absent",
		"--property",
		"foo=bar",
		"--fail-threshold",
		"0",
		"--changes",
		"--send-report",
		"--analysis-id",
		"id",
	}, " ")
	actual := strings.Join(core.GetCmdOptions(testOptions), " ")
	if expected != actual {
		t.Fatalf("expected \"%s\" got \"%s\"", expected, actual)
	}
}

func TestAllCommands(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil || (runtime.GOOS == "windows" && isGitHubAction()) {
		t.Skip(err)
	}
	core.DisableColor()
	core.CheckForUpdates("0.1.0")
	resultsPath := "/tmp/qodana_scan_results"
	err := os.MkdirAll(resultsPath, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	projectPath := "/tmp/qodana_scan"
	err = os.MkdirAll(projectPath, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(projectPath, "hello.py"), []byte("println(\"Hello\")\n123"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// pull
	out := bytes.NewBufferString("")
	command := NewPullCommand()
	command.SetOut(out)
	command.SetArgs([]string{"-i", projectPath})
	err = command.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// scan
	out = bytes.NewBufferString("")
	command = NewScanCommand()
	command.SetOut(out)
	command.SetArgs([]string{
		"-i", projectPath,
		"-o", resultsPath,
		"--fail-threshold", "5",
		"--print-problems",
		"--clear-cache",
	})
	err = command.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// view
	out = bytes.NewBufferString("")
	command = NewViewCommand()
	command.SetOut(out)
	command.SetArgs([]string{"-f", filepath.Join(resultsPath, "qodana.sarif.json")})
	err = command.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// show
	out = bytes.NewBufferString("")
	command = NewShowCommand()
	command.SetOut(out)
	command.SetArgs([]string{"-i", projectPath, "-d"})
	err = command.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// init after project analysis with .idea inside
	err = os.Remove(filepath.Join(projectPath, "qodana.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	out = bytes.NewBufferString("")
	command = NewInitCommand()
	command.SetOut(out)
	command.SetArgs([]string{"-i", projectPath})
	err = command.Execute()
	if err != nil {
		t.Fatal(err)
	}

	err = os.RemoveAll(resultsPath)
	if err != nil {
		t.Fatal(err)
	}
	err = os.RemoveAll(projectPath)
	if err != nil {
		t.Fatal(err)
	}
}
