package main

// Looking for the marks in a file, and keeping each identifier once with
// every place it turned up.

import (
	"fmt"
	"sort"
	"strings"
)

// A sighting is one identifier and everywhere it was seen.
type sighting struct {
	mark   *mark
	found  found
	places []string
}

// key is what makes two sightings the same one: the mark and the value,
// so the same token in forty bundles is one finding with forty places.
func (s *sighting) key() string {
	return s.mark.key + "\x00" + s.found.value
}

type side struct {
	seen  map[string]*sighting
	order []string
}

func newSide() *side {
	return &side{seen: map[string]*sighting{}}
}

func (s *side) add(m *mark, f found, place string) {
	k := m.key + "\x00" + f.value
	sg, ok := s.seen[k]
	if !ok {
		sg = &sighting{mark: m, found: f}
		s.seen[k] = sg
		s.order = append(s.order, k)
	}
	for _, p := range sg.places {
		if p == place {
			return
		}
	}
	if len(sg.places) < 200 {
		sg.places = append(sg.places, place)
	}
}

func (s *side) sightings() []*sighting {
	out := make([]*sighting, 0, len(s.order))
	for _, k := range s.order {
		out = append(out, s.seen[k])
	}
	return out
}

// scanText looks for every mark in one file's text. where is how the file
// is shown, name what it is called, which some marks care about. located
// is false for the strings pulled out of a binary, where a line number
// would be a line of that and nothing anybody could find. borrowed is
// set for text that is other people's by its nature: the strings of a
// binary, which has its libraries compiled in, and licence files.
func scanText(s *side, where, name, text string, located, borrowed bool) {
	var lines []int
	for _, m := range marks {
		if !m.applies(name) || !hinted(text, m.hints) || borrowed && (m.links || m.key == "email") {
			continue
		}
		for _, idx := range m.re.FindAllStringSubmatchIndex(text, -1) {
			groups := make([]string, len(idx)/2)
			for g := range groups {
				if idx[2*g] >= 0 {
					groups[g] = text[idx[2*g]:idx[2*g+1]]
				}
			}
			f, ok := m.read(groups)
			if !ok || common(f) {
				continue
			}
			if !located {
				s.add(m, f, where)
				continue
			}
			if lines == nil {
				lines = lineStarts(text)
			}
			s.add(m, f, where+", "+position(lines, idx[0]))
		}
	}
}

func common(f found) bool {
	for _, t := range f.ties {
		if commonTies[t] {
			return true
		}
	}
	return false
}

func hinted(text string, hints []string) bool {
	if len(hints) == 0 {
		return true
	}
	for _, h := range hints {
		if strings.Contains(text, h) {
			return true
		}
	}
	return false
}

func lineStarts(text string) []int {
	starts := []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// position is a line, and in a long line a column too: a minified bundle
// is one line, and "line 1" alone says nothing.
func position(starts []int, at int) string {
	line := sort.Search(len(starts), func(i int) bool { return starts[i] > at }) - 1
	col := at - starts[line] + 1
	if col > 200 {
		return fmt.Sprintf("line %s, column %s", commas(line+1), commas(col))
	}
	return fmt.Sprintf("line %s", commas(line+1))
}
