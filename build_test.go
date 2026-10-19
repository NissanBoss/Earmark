package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tokens and keys in these tests are put together from pieces, so that no
// literal in the repository has the shape of a real one. GitHub refuses a
// push with something that looks like a live key in it, made up or not.

func b64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func mapboxToken(user string) string {
	body, _ := json.Marshal(map[string]string{"u": user, "a": "ckx12ab3c0001"})
	return "pk" + "." + base64.RawURLEncoding.EncodeToString(body) + ".Qx7Kp2LmN9vR4sT1uW3yZ5"
}

func jwt(claims map[string]any) string {
	head, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	body, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(head) + "." + base64.RawURLEncoding.EncodeToString(body) + ".c2lnbmF0dXJlc2lnbmF0dXJl"
}

// project writes files into a folder of the test's own.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// check reads one project, and another to compare it with if given.
func check(t *testing.T, mine string, other ...string) *run {
	t.Helper()
	c := newRun(shower{})
	for i, paths := range [][]string{{mine}, other} {
		if len(paths) == 0 {
			continue
		}
		files, skipped, err := filesIn(paths, c.show)
		if err != nil {
			t.Fatal(err)
		}
		c.skipped[i] = skipped
		for _, f := range files {
			c.file(i, f)
		}
	}
	return c
}

func (c *run) find(key string) []*sighting {
	var out []*sighting
	for _, sg := range c.sides[0].sightings() {
		if sg.mark.key == key {
			out = append(out, sg)
		}
	}
	return out
}

func (c *run) only(t *testing.T, keys ...string) {
	t.Helper()
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	got := map[string]bool{}
	for _, sg := range c.sides[0].sightings() {
		got[sg.mark.key] = true
		if !want[sg.mark.key] {
			t.Errorf("found %s (%s), which should not be there", sg.mark.key, sg.found.value)
		}
	}
	for k := range want {
		if !got[k] {
			t.Errorf("expected %s, found none", k)
		}
	}
}

func (c *run) text(compare bool) string {
	var b bytes.Buffer
	c.report(compare).render(&b)
	return b.String()
}

func zipOf(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write(body)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func contains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected %q in:\n%s", needle, haystack)
	}
}
