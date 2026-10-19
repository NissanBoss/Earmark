package main

// The calibration: a thousand personal sites, many of them with a second
// site by the same person, the most visited sites on the web, and Android
// apps their developers publish on GitHub, downloaded by the scripts in
// testdata on GitHub's machines and counted. It is where the numbers in
// the README come from, and where a rule that fires on what is not
// anybody's shows up: a theme's author, a library's credit, a sign-up
// form's own. Only counts are printed, and identifiers masked; the sites
// are other people's.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func scanned(t *testing.T, path string) *side {
	t.Helper()
	c := newRun(shower{})
	files, _, err := filesIn([]string{path}, c.show)
	if err != nil {
		return c.sides[0]
	}
	for _, f := range files {
		c.file(0, f)
	}
	return c.sides[0]
}

func tiesOf(s *side) map[string]string {
	out := map[string]string{}
	for _, sg := range s.sightings() {
		for _, t := range sg.found.ties {
			out[t] = sg.mark.key
		}
	}
	return out
}

func TestCalibration(t *testing.T) {
	dir := os.Getenv("EARMARK_CALIBRATION")
	if dir == "" {
		t.Skip("EARMARK_CALIBRATION is not set; see testdata/sites.py, apks.py and tranco.py")
	}
	if _, err := os.Stat(filepath.Join(dir, "sites")); err == nil {
		calibrateSites(t, dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "apks")); err == nil {
		calibrateApps(t, filepath.Join(dir, "apks"))
	}
	if _, err := os.Stat(filepath.Join(dir, "tranco")); err == nil {
		calibrateWeb(t, filepath.Join(dir, "tranco"))
	}
}

func calibrateSites(t *testing.T, dir string) {
	entries, _ := os.ReadDir(filepath.Join(dir, "sites"))
	var owners []string
	sides := map[string]*side{}
	byMark := map[string]int{}
	named, githubOwner, githubOther := 0, 0, 0
	seenBy := map[string]map[string]bool{}
	for _, e := range entries {
		o := e.Name()
		owners = append(owners, o)
		s := scanned(t, filepath.Join(dir, "sites", o))
		sides[o] = s
		any := false
		marksHere := map[string]bool{}
		for _, sg := range s.sightings() {
			marksHere[sg.mark.key] = true
			if sg.found.level == names {
				any = true
			}
			if sg.mark.key == "github" || sg.mark.key == "github-link" || sg.mark.key == "github-pages" {
				if strings.EqualFold(sg.found.who, o) {
					githubOwner++
				} else {
					githubOther++
				}
			}
			for _, tie := range sg.found.ties {
				if seenBy[tie] == nil {
					seenBy[tie] = map[string]bool{}
				}
				seenBy[tie][o] = true
			}
		}
		for k := range marksHere {
			byMark[k]++
		}
		if any {
			named++
		}
	}
	t.Logf("personal sites: %d read, %d with something that names an account", len(owners), named)
	t.Logf("GitHub accounts found: %d the site's owner, %d somebody else", githubOwner, githubOther)
	logCounts(t, "sites carrying each kind", byMark)

	// What many owners share is a theme's or a library's, not a person's.
	type common struct {
		tie string
		n   int
	}
	var commons []common
	for tie, by := range seenBy {
		if len(by) >= 4 {
			commons = append(commons, common{tie, len(by)})
		}
	}
	sort.Slice(commons, func(i, j int) bool {
		return commons[i].n > commons[j].n || commons[i].n == commons[j].n && commons[i].tie < commons[j].tie
	})
	for _, c := range commons[:min(len(commons), 120)] {
		t.Logf("seen on %3d sites: %s", c.n, c.tie)
	}

	// Two sites of one person, and two of different people.
	pairs, tied, tiedBeyondHost := 0, 0, 0
	families := map[string]int{}
	for _, o := range owners {
		p := filepath.Join(dir, "projects", o)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		pairs++
		mine, theirs := tiesOf(sides[o]), tiesOf(scanned(t, p))
		hit, beyond := false, false
		for tie := range mine {
			if _, ok := theirs[tie]; ok {
				hit = true
				if tie != "github:"+strings.ToLower(o) {
					beyond = true
					families[strings.SplitN(tie, ":", 2)[0]]++
				}
			}
		}
		if hit {
			tied++
		}
		if beyond {
			tiedBeyondHost++
		}
	}
	t.Logf("two sites of one owner: %d pairs, %d tied, %d tied by something besides the GitHub account they are both served from", pairs, tied, tiedBeyondHost)
	logCounts(t, "what tied them, besides the account", families)

	strangers, falseTies := 0, 0
	wrong := map[string]int{}
	for i := range owners {
		a, b := owners[i], owners[(i+1)%len(owners)]
		if strings.EqualFold(a, b) {
			continue
		}
		strangers++
		mine, theirs := tiesOf(sides[a]), tiesOf(sides[b])
		hit := false
		for tie := range mine {
			if _, ok := theirs[tie]; ok {
				hit = true
				wrong[maskTie(tie)]++
			}
		}
		if hit {
			falseTies++
		}
	}
	t.Logf("two sites of different owners: %d pairs, %d tied", strangers, falseTies)
	logCounts(t, "what tied different owners", wrong)
}

func calibrateApps(t *testing.T, dir string) {
	files, _ := filepath.Glob(filepath.Join(dir, "*.apk"))
	read, person, city, debug, empty := 0, 0, 0, 0, 0
	byKey := map[string]map[string]bool{}
	byMark := map[string]int{}
	for _, f := range files {
		s := scanned(t, f)
		owner := strings.SplitN(filepath.Base(f), "__", 2)[0]
		signed := false
		marksHere := map[string]bool{}
		for _, sg := range s.sightings() {
			marksHere[sg.mark.key] = true
			if sg.mark.key != "signing" {
				continue
			}
			signed = true
			switch {
			case strings.Contains(sg.found.note, "debug key"):
				debug++
			case strings.Contains(sg.found.who, "name ") || strings.Contains(sg.found.who, "organisation "):
				person++
				if strings.Contains(sg.found.who, "city ") {
					city++
				}
			default:
				empty++
			}
			if byKey[sg.found.value] == nil {
				byKey[sg.found.value] = map[string]bool{}
			}
			byKey[sg.found.value][strings.ToLower(owner)] = true
		}
		for k := range marksHere {
			byMark[k]++
		}
		if signed {
			read++
		}
	}
	shared := 0
	for _, owners := range byKey {
		if len(owners) > 1 {
			shared++
		}
	}
	logCommon(t, "apps", dir, files)
	t.Logf("apps: %d of %d with a certificate read; %d name somebody, %d of them with a city; %d signed with a debug key; %d with nothing in the name", read, len(files), person, city, debug, empty)
	t.Logf("keys that sign apps under more than one GitHub account: %d", shared)
	logCounts(t, "apps carrying each kind", byMark)
}

func calibrateWeb(t *testing.T, dir string) {
	entries, _ := os.ReadDir(dir)
	byMark := map[string]int{}
	seenOn := map[string]map[string]bool{}
	for _, e := range entries {
		s := scanned(t, filepath.Join(dir, e.Name()))
		marksHere := map[string]bool{}
		for _, sg := range s.sightings() {
			marksHere[sg.mark.key] = true
			for _, tie := range sg.found.ties {
				if seenOn[tie] == nil {
					seenOn[tie] = map[string]bool{}
				}
				seenOn[tie][registrable(e.Name())] = true
			}
		}
		for k := range marksHere {
			byMark[k]++
		}
	}
	t.Logf("most visited sites: %d read", len(entries))
	var paths []string
	for _, e := range entries {
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	logCommon(t, "most visited sites", dir, paths)
	logCounts(t, "sites carrying each kind", byMark)
	families := map[string]int{}
	for tie, on := range seenOn {
		if len(on) > 1 {
			families[strings.SplitN(tie, ":", 2)[0]]++
		}
	}
	logCounts(t, "identifiers on more than one of them", families)
}

func maskTie(tie string) string {
	family, value, _ := strings.Cut(tie, ":")
	return family + ":" + maskID(value)
}

func logCounts(t *testing.T, title string, counts map[string]int) {
	var keys []string
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return counts[keys[i]] > counts[keys[j]] || counts[keys[i]] == counts[keys[j]] && keys[i] < keys[j]
	})
	var parts []string
	for _, k := range keys[:min(len(keys), 60)] {
		parts = append(parts, fmt.Sprintf("%s %d", k, counts[k]))
	}
	t.Logf("%s: %s", title, strings.Join(parts, ", "))
}

// logCommon prints what turns up in many unrelated projects, which is a
// library's or a vendor's and not anybody's, so it can be told apart. These
// are not one person's, by the way they were found, and are printed as
// they are.
func logCommon(t *testing.T, what, dir string, paths []string) {
	seen := map[string]map[string]bool{}
	for _, p := range paths {
		owner := strings.SplitN(filepath.Base(p), "__", 2)[0]
		for _, sg := range scanned(t, p).sightings() {
			if sg.found.level != names && sg.found.level != ties {
				continue
			}
			for _, tie := range sg.found.ties {
				if seen[tie] == nil {
					seen[tie] = map[string]bool{}
				}
				seen[tie][owner] = true
			}
		}
	}
	type common struct {
		tie string
		n   int
	}
	var list []common
	for tie, by := range seen {
		if len(by) >= 3 {
			list = append(list, common{tie, len(by)})
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].n > list[j].n || list[i].n == list[j].n && list[i].tie < list[j].tie
	})
	for _, c := range list[:min(len(list), 80)] {
		t.Logf("%s, in %d of them: %s", what, c.n, c.tie)
	}
}
