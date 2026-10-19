package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Projects made by the tools people use, built by testdata/tools.sh, each
// with what must be found in it. For the keys, what must be found is the
// fingerprint the tool that made the key prints for it: the comparison is
// with ssh-keygen and gpg, not with this program's own idea of the answer.
func TestMadeByTools(t *testing.T) {
	dir := os.Getenv("EARMARK_TOOLS")
	if dir == "" {
		t.Skip("EARMARK_TOOLS is not set; see testdata/tools.sh")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "expect.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			t.Fatalf("a line of expect.tsv is not three columns: %q", line)
		}
		path, key, want := parts[0], parts[1], parts[2]
		t.Run(path+" "+key, func(t *testing.T) {
			s := scanned(t, filepath.Join(dir, filepath.FromSlash(path)))
			var seen []string
			for _, sg := range s.sightings() {
				if sg.mark.key != key {
					continue
				}
				f := sg.found
				if strings.Contains(f.who, want) || strings.Contains(f.value, want) || strings.Contains(f.note, want) {
					return
				}
				seen = append(seen, f.value+" / "+f.who)
			}
			t.Errorf("no %s with %q in %s; found %v", key, want, path, seen)
		})
	}
}
