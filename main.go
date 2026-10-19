// Earmark says what ties a project to you: which identifiers in it belong
// to an account of yours, which of them say outright whose account it is,
// and which it shares with another project of yours.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
)

var version = "unreleased"

const (
	exitClean   = 0
	exitFound   = 1 // something names you, is listed, or is shared
	exitBroken  = 2
	exitNothing = 3 // nothing could be read, which is not the same as nothing found
)

const tooBig = 512 << 20

func main() {
	var mine, other []string
	reveal, against := false, false
	for _, a := range os.Args[1:] {
		switch a {
		case "--reveal", "-reveal":
			reveal = true
		case "--version", "-version":
			fmt.Println("Earmark " + version)
			return
		case "--against", "-against":
			against = true
		case "-h", "--help", "-help":
			usage()
			return
		default:
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				fmt.Fprintf(os.Stderr, "earmark: %s is not an option this has\n", a)
				usage()
				os.Exit(exitBroken)
			}
			if against {
				other = append(other, a)
			} else {
				mine = append(mine, a)
			}
		}
	}
	if len(mine) == 0 || against && len(other) == 0 {
		usage()
		os.Exit(exitBroken)
	}
	home, _ := os.UserHomeDir()
	show := shower{home: home, reveal: reveal}
	// An app is mostly pictures this does not read, but a bundle of
	// a few hundred megabytes is still read whole; this keeps it from ever
	// being what slows a machine down.
	debug.SetMemoryLimit(1 << 30)

	c := newRun(show)
	for i, paths := range [][]string{mine, other} {
		if len(paths) == 0 {
			continue
		}
		files, skipped, err := filesIn(paths, show)
		if err != nil {
			fmt.Fprintf(os.Stderr, "earmark: %v\n", err)
			os.Exit(exitBroken)
		}
		c.skipped[i] = skipped
		for _, f := range files {
			c.file(i, f)
		}
	}
	r := c.report(len(other) > 0)
	r.render(os.Stdout)
	switch {
	case c.read[0] == 0:
		os.Exit(exitNothing)
	case r.worst() >= Listed:
		os.Exit(exitFound)
	}
	os.Exit(exitClean)
}

// A run holds the two projects: the one being checked, and the other one
// it is compared with, if there is one.
type run struct {
	show       shower
	sides      [2]*side
	gaps       []string
	read       [2]int
	containers [2]int
	inside     [2]int
	skipped    [2]map[string]int
}

func newRun(show shower) *run {
	return &run{show: show, sides: [2]*side{newSide(), newSide()}}
}

func (c *run) gap(where, why string) {
	c.gaps = append(c.gaps, where+": "+why)
}

func (c *run) file(i int, path string) {
	where := c.show.path(path)
	info, err := os.Stat(path)
	if err != nil {
		c.gap(where, "could not be opened")
		return
	}
	if info.Size() > tooBig {
		c.gap(where, fmt.Sprintf("is %s, more than this reads", size(info.Size())))
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		c.gap(where, "could not be read")
		return
	}
	c.read[i]++
	c.inspect(i, where, filepath.Base(path), data, 0)
}

// inspect reads one file, or one file inside another. depth stops a zip
// inside a jar inside an APK from going on for ever.
func (c *run) inspect(i int, where, name string, data []byte, depth int) {
	s := c.sides[i]
	lower := strings.ToLower(name)
	if depth > 0 {
		c.inside[i]++
	}
	switch {
	case strings.HasSuffix(lower, ".mobileprovision"):
		if !c.profile(s, where, data) {
			c.gap(where, "is not a provisioning profile this can read")
		}
	case bytes.HasPrefix(data, []byte("PK\x03\x04")), bytes.HasPrefix(data, []byte("Cr24")):
		if depth >= 3 {
			c.gap(where, "is a container inside a container inside a container, and was not opened")
			return
		}
		entries, cd, gaps, err := zipEntries(data)
		if err != nil {
			c.gap(where, "looks like a zip and could not be opened as one")
			return
		}
		c.containers[i]++
		for _, g := range gaps {
			c.gap(where, g)
		}
		if cd > 0 {
			for _, der := range apkSigningBlock(data, cd) {
				if f, ok := certFound(der); ok {
					s.add(signingMark, f, where+", in its signing block")
				}
			}
		}
		for _, e := range entries {
			inner := where + " > " + e.name
			if l := strings.ToLower(e.name); strings.HasPrefix(l, "meta-inf/") && (strings.HasSuffix(l, ".rsa") || strings.HasSuffix(l, ".dsa") || strings.HasSuffix(l, ".ec")) {
				for _, der := range pkcs7Certs(e.data) {
					if f, ok := certFound(der); ok {
						s.add(signingMark, f, inner)
					}
				}
				continue
			}
			c.inspect(i, inner, e.name, e.data, depth+1)
		}
	case bytes.HasPrefix(data, []byte{0x1f, 0x8b}), looksTar(data):
		if depth >= 3 {
			c.gap(where, "is a container inside a container inside a container, and was not opened")
			return
		}
		entries, gaps, err := tarEntries(data)
		if err != nil {
			c.gap(where, "could not be unpacked")
			return
		}
		c.containers[i]++
		for _, g := range gaps {
			c.gap(where, g)
		}
		for _, e := range entries {
			c.inspect(i, where+" > "+e.name, e.name, e.data, depth+1)
		}
	case isText(data):
		text := string(data)
		scanText(s, where, name, text, true, depth > 0 && licence(name))
		scanPGP(s, where, text)
	default:
		// A binary keeps its strings as runs of text, and a token baked
		// into an app is one of them. Where in the file is of no use to
		// anybody, so only the file is named.
		if bytes.Contains(data[:min(len(data), 4096)], []byte("<?xml")) && bytes.Contains(data, []byte("TeamIdentifier")) {
			c.profile(s, where, data)
			return
		}
		scanText(s, where+", in its strings", name, strs(data), false, true)
	}
}

// licence is a file inside a package that is about its libraries: the
// licences and notices, and the metadata a jar keeps under META-INF.
func licence(name string) bool {
	l := strings.ToLower(name)
	base := l[strings.LastIndex(l, "/")+1:]
	return strings.HasPrefix(l, "meta-inf/") || strings.Contains(l, "/meta-inf/") ||
		strings.Contains(base, "license") || strings.Contains(base, "licence") || strings.Contains(base, "notice") ||
		strings.Contains(l, "third_party") || strings.Contains(l, "third-party") || strings.Contains(l, "open_source_licenses")
}

func (c *run) report(compare bool) *Report {
	r := &Report{Title: "Earmark: what ties this to you"}
	r.Summary = append(r.Summary, c.summary(compare)...)
	r.Gaps = append(r.Gaps, c.gaps...)

	if compare {
		if f, ok := c.shared(); ok {
			r.add(f)
		}
	}
	groups := map[string][]*sighting{}
	var order []string
	for _, sg := range c.sides[0].sightings() {
		k := fmt.Sprintf("%d\x00%s", 9-int(sg.found.level), sg.mark.key)
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], sg)
	}
	sort.SliceStable(order, func(a, b int) bool { return order[a] < order[b] })
	for _, k := range order {
		r.add(c.finding(groups[k]))
	}
	if len(r.Findings) == 0 && c.read[0] > 0 && len(r.Gaps) == 0 {
		r.add(Finding{Tag: "CLEAR   ", Weight: Note,
			Title:  "Nothing here ties it to an account",
			Advice: "That covers the identifiers this knows, listed in its README. A project can still be tied to you by how it is written, by when it changes and by who links to it, none of which is looked at here."})
	}
	return r
}

func (c *run) summary(compare bool) []string {
	var out []string
	line := "Looked at " + plural(c.read[0], "file", "files")
	if c.containers[0] > 0 {
		line += fmt.Sprintf(", %s with %s inside", plural(c.containers[0], "of them a container", "of them containers"), plural(c.inside[0], "file", "files"))
	}
	line += "."
	if compare {
		line += fmt.Sprintf(" And at %s of the other project, for what the two share.", plural(c.read[1]+c.inside[1], "file", "files"))
	}
	out = append(out, line)
	skipped := map[string]int{}
	for _, m := range c.skipped {
		for k, v := range m {
			skipped[k] += v
		}
	}
	if skipped["node_modules"] > 0 || skipped["bower_components"] > 0 || skipped["Pods"] > 0 {
		out = append(out, "Libraries fetched by a package manager (node_modules, Pods) were not read: the identifiers in them are their authors', not yours. A built bundle has the same libraries in it, and is read.")
	}
	if skipped[".git"] > 0 {
		out = append(out, "The .git folder was not read. What a repository's history says about you is a question of its own.")
	}
	return out
}

func (c *run) finding(group []*sighting) Finding {
	m := group[0].mark
	lv := group[0].found.level
	plain := true
	var lines []string
	for n, sg := range group {
		if n > 0 {
			lines = append(lines, "")
		}
		f := sg.found
		var head string
		at := strings.Index(strings.ToLower(f.value), strings.ToLower(f.who))
		switch {
		case f.who != "" && at >= 0:
			// The identifier is the name, or the name is in it as it
			// stands: a GitHub account, ko-fi.com/ and a name, an address.
			head = c.show.id(f.value[:at]) + c.show.text(f.value[at:at+len(f.who)]) + c.show.id(f.value[at+len(f.who):])
		case f.who != "":
			head = c.show.id(f.value) + ", which names " + c.show.text(f.who)
			plain = false
		default:
			head = c.show.id(f.value)
			plain = false
		}
		lines = append(lines, head)
		if f.note != "" {
			for _, l := range wrapText(f.note, wrap-2) {
				lines = append(lines, "  "+l)
			}
		}
		for _, p := range atMost(sg.places, 3) {
			lines = append(lines, "  "+p)
		}
	}
	lines = append(lines, "")
	lines = append(lines, wrapText(m.lookup, wrap)...)
	return Finding{Tag: tags[lv], Weight: weights[lv], Title: title(m.name, lv, len(group), plain), Lines: lines, Advice: m.advice}
}

var tags = map[level]string{names: "NAMES   ", listed: "LISTED  ", ties: "TIES    "}
var weights = map[level]Severity{names: Names, listed: Listed, ties: Ties}

// title says what was found in the words of the level: an identifier that
// is a name, one that gives a name away, one the lookup services index,
// and one that only matches.
func title(name string, lv level, n int, plain bool) string {
	what := article(name)
	if n > 1 {
		what = fmt.Sprintf("%d %s", n, pluralName(name))
	}
	switch {
	case lv == names && plain:
		return "It carries " + what + " by name"
	case lv == names && n == 1:
		return "It carries " + what + " that names its account"
	case lv == names:
		return "It carries " + what + " that name their accounts"
	case lv == listed && n == 1:
		return "It carries " + what + ", which the lookup services index"
	case lv == listed:
		return "It carries " + what + ", which the lookup services index"
	case n == 1:
		return "It carries " + what + ", the same in everything else of its account"
	}
	return "It carries " + what + ", each the same in everything else of its account"
}

func article(name string) string {
	if strings.ContainsRune("aeiouAEIOU", rune(name[0])) {
		return "an " + name
	}
	return "a " + name
}

func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func pluralName(name string) string {
	switch {
	case strings.HasSuffix(name, "s"):
		return name + "es"
	case strings.HasSuffix(name, "ty"):
		return name[:len(name)-1] + "ies"
	}
	return name + "s"
}

func usage() {
	fmt.Fprintln(os.Stderr, `Earmark - what ties a project to you

  earmark dist/                          a build, a folder, an app or a package
  earmark side-project/ --against main/  and what it shares with another of yours
  earmark --reveal ...                   show the names and identifiers in full
  earmark --version

A public token, an analytics tag, the address a site loads its pictures
from: none of them is a secret, so nothing checks them, and each belongs
to an account. Some say outright whose: a Mapbox token has the user name
inside it. Some are listed by services that will give anybody every other
site that carries them. This says which ones a project carries, and with
--against, which it shares with another project of yours.

It only reads. Nothing is written, and nothing is looked up or sent
anywhere: every answer comes from the files themselves.`)
}
