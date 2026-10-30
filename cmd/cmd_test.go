// Copyright 2026 Google LLC
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

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/keep-sorted/keepsorted"
)

func TestAllowNoFiles(t *testing.T) {
	dir := t.TempDir()
	// Split markers so keep-sorted does not treat test fixtures as directives.
	const sorted = "// keep-" + "sorted start\na\nb\n// keep-" + "sorted end\n"
	const unsorted = "// keep-" + "sorted start\nb\na\n// keep-" + "sorted end\n"
	file := filepath.Join(dir, "sorted.txt")
	if err := os.WriteFile(file, []byte(sorted), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.txt")

	for _, mode := range []string{"fix", "lint"} {
		for _, tc := range []struct {
			name         string
			allowNoFiles bool
			invalidID    bool
			files        []string
			stdin        string
			wantOK       bool
			wantStdout   string
			wantError    string
			wantFindings bool
		}{
			{name: "default rejects no files", wantError: "must pass one or more filenames"},
			{name: "enabled accepts no files", allowNoFiles: true, wantOK: true},
			{name: "missing file still fails", allowNoFiles: true, files: []string{missing}, wantError: "missing.txt"},
			{name: "configuration still validated", allowNoFiles: true, invalidID: true, wantError: "id cannot be empty"},
			{name: "named file still processed", allowNoFiles: true, files: []string{file}, wantOK: true},
			{name: "stdin still processed", allowNoFiles: true, files: []string{"-"}, stdin: unsorted, wantOK: mode == "fix", wantStdout: sorted, wantFindings: mode == "lint"},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				c := &Config{
					id:             "keep-sorted",
					defaultOptions: keepsorted.DefaultBlockOptions(),
					operation:      operations[mode],
					allowNoFiles:   tc.allowNoFiles,
				}
				if tc.invalidID {
					c.id = ""
				}
				input, err := os.CreateTemp(t.TempDir(), "stdin")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { input.Close() })
				if _, err := input.WriteString(tc.stdin); err != nil {
					t.Fatal(err)
				}
				if _, err := input.Seek(0, 0); err != nil {
					t.Fatal(err)
				}
				output, err := os.CreateTemp(t.TempDir(), "stdout")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { output.Close() })
				// Standard streams are shared, so these subtests must remain sequential.
				previousStdin, previousStdout := os.Stdin, os.Stdout
				t.Cleanup(func() { os.Stdin, os.Stdout = previousStdin, previousStdout })
				os.Stdin, os.Stdout = input, output

				ok, err := Run(c, tc.files)
				if ok != tc.wantOK {
					t.Errorf("Run() ok = %v, want %v", ok, tc.wantOK)
				}
				if tc.wantError == "" {
					if err != nil {
						t.Errorf("Run() error = %v", err)
					}
				} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Errorf("Run() error = %v, want error containing %q", err, tc.wantError)
				}
				stdout, err := os.ReadFile(output.Name())
				if err != nil {
					t.Fatal(err)
				}
				if tc.wantFindings {
					if !strings.Contains(string(stdout), "These lines are out of order.") {
						t.Errorf("expected sorting finding in stdout, got %q", stdout)
					}
				} else if string(stdout) != tc.wantStdout {
					t.Errorf("stdout = %q, want %q", stdout, tc.wantStdout)
				}
			})
		}
	}
}
