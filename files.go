package main

// Which files are read, how, and how their paths are shown.

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Text read as text: what a site, an app's sources or a repository is made
// of. Anything else in a folder is passed over, so pointing this at a
// project does not mean reading every picture in it.
var textExt = map[string]bool{
	".html": true, ".htm": true, ".xhtml": true, ".js": true, ".mjs": true, ".cjs": true, ".jsx": true,
	".ts": true, ".tsx": true, ".vue": true, ".svelte": true, ".astro": true, ".css": true, ".scss": true,
	".json": true, ".map": true, ".xml": true, ".plist": true, ".txt": true, ".md": true, ".mdx": true,
	".markdown": true, ".yml": true, ".yaml": true, ".toml": true, ".ini": true, ".cfg": true, ".conf": true,
	".env": true, ".properties": true, ".gradle": true, ".kts": true, ".swift": true, ".kt": true,
	".java": true, ".m": true, ".mm": true, ".h": true, ".py": true, ".rb": true, ".php": true, ".go": true,
	".rs": true, ".cs": true, ".dart": true, ".pbxproj": true, ".xcconfig": true, ".entitlements": true,
	".webmanifest": true, ".svg": true, ".asc": true, ".pub": true, ".gpg": true, ".liquid": true,
	".njk": true, ".hbs": true, ".ejs": true, ".erb": true, ".twig": true, ".rst": true, ".adoc": true,
	".tf": true, ".tfvars": true, ".sh": true, ".ps1": true, ".lock": true, ".csv": true, ".tsv": true,
}

// Files with no extension that are text and say something.
var textNames = map[string]bool{
	"cname": true, "dockerfile": true, "procfile": true, ".firebaserc": true, "authorized_keys": true,
	"readme": true, "humans.txt": true, ".npmrc": true, "funding.yml": true, "podfile": true, "gemfile": true,
}

// Containers read inside without unpacking anything.
var archiveExt = map[string]bool{
	".zip": true, ".apk": true, ".aab": true, ".ipa": true, ".xpi": true, ".crx": true, ".jar": true,
	".whl": true, ".vsix": true, ".nupkg": true, ".tgz": true, ".gz": true, ".tar": true, ".war": true,
	".mobileprovision": true, ".appx": true, ".msix": true,
}

// Folders that are other people's code, or a machine's leftovers. What is
// in node_modules was written by the authors of the libraries, and every
// identifier in it would be a library's, reported as yours.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "bower_components": true, ".venv": true, "venv": true,
	"__pycache__": true, "Pods": true, ".gradle": true, ".idea": true,
}

func worthOpening(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return textExt[ext] || archiveExt[ext] || textNames[strings.ToLower(filepath.Base(name))]
}

// filesIn lists what is under the paths given. A file named on the
// command line is always opened, whatever it is called. The error names
// the path the way the report would, home folder as ~.
func filesIn(paths []string, show shower) (files []string, skipped map[string]int, err error) {
	skipped = map[string]int{}
	for _, root := range paths {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, nil, err
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: no such file or folder, or it cannot be opened", show.path(abs))
		}
		if !info.IsDir() {
			files = append(files, abs)
			continue
		}
		err = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if p != abs && skipDirs[d.Name()] {
					skipped[d.Name()]++
					return fs.SkipDir
				}
				return nil
			}
			if d.Type().IsRegular() && worthOpening(p) {
				files = append(files, p)
			} else if d.Type().IsRegular() {
				skipped[""]++
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	sort.Strings(files)
	return files, skipped, nil
}

// isText says whether a file reads as text: no NUL in the first stretch
// of it, and valid UTF-8 there, give or take a character cut in two at the
// end of the stretch.
func isText(b []byte) bool {
	head := b[:min(len(b), 8192)]
	if bytes.IndexByte(head, 0) >= 0 {
		return false
	}
	for cut := 0; cut < 4 && len(head) > 0; cut++ {
		if utf8.Valid(head) {
			return true
		}
		if len(head) == len(b) {
			return false
		}
		head = head[:len(head)-1]
	}
	return false
}

// strs is the text inside a binary: runs of printable ASCII, and runs of
// UTF-16, which is how Android's resources and Windows binaries keep
// their strings. Each run is a line of its own, so a pattern never joins
// two of them.
func strs(b []byte) string {
	var out strings.Builder
	run := 0
	start := 0
	flush := func(end int) {
		if run >= 6 {
			out.Write(b[start:end])
			out.WriteByte('\n')
		}
		run = 0
	}
	for i, c := range b {
		if c >= 0x20 && c < 0x7f {
			if run == 0 {
				start = i
			}
			run++
			continue
		}
		flush(i)
	}
	flush(len(b))
	// UTF-16 little-endian: a printable byte, a zero, and again, from an
	// even byte and from an odd one, as nothing says where a string starts.
	for shift := 0; shift < 2; shift++ {
		run = 0
		var wide []byte
		for i := shift; i+1 < len(b); i += 2 {
			if b[i] >= 0x20 && b[i] < 0x7f && b[i+1] == 0 {
				wide = append(wide, b[i])
				run++
				continue
			}
			if run >= 6 {
				out.Write(wide)
				out.WriteByte('\n')
			}
			wide, run = wide[:0], 0
		}
		if run >= 6 {
			out.Write(wide)
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// entry is a file inside a container.
type entry struct {
	name string
	data []byte
}

const maxEntry = 64 << 20

// zipEntries reads the files inside a zip, an APK or an IPA. An entry that
// is encrypted or broken is a gap, said, not skipped.
func zipEntries(data []byte) (entries []entry, cdOffset int, gaps []string, err error) {
	// A Chrome extension is a zip with a header in front of it.
	if bytes.HasPrefix(data, []byte("Cr24")) {
		if i := bytes.Index(data, []byte("PK\x03\x04")); i > 0 {
			data = data[i:]
		}
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, 0, nil, err
	}
	cdOffset = centralDirectory(data)
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !worthInside(f.Name) {
			continue
		}
		if f.Flags&1 != 0 {
			gaps = append(gaps, f.Name+" is encrypted")
			continue
		}
		if f.UncompressedSize64 > maxEntry {
			gaps = append(gaps, fmt.Sprintf("%s is %s, more than this reads inside a container", f.Name, size(int64(f.UncompressedSize64))))
			continue
		}
		rc, err := f.Open()
		if err != nil {
			gaps = append(gaps, f.Name+" could not be opened")
			continue
		}
		body, err := io.ReadAll(io.LimitReader(rc, maxEntry+1))
		rc.Close()
		if err != nil {
			gaps = append(gaps, f.Name+" could not be read")
			continue
		}
		entries = append(entries, entry{f.Name, body})
	}
	return entries, cdOffset, gaps, nil
}

// worthInside picks what to read in a container: text, the signatures,
// and the binaries that keep an app's strings. Pictures and sounds in an
// app are most of its size and say nothing here.
func worthInside(name string) bool {
	lower := strings.ToLower(name)
	base := path.Base(lower)
	ext := path.Ext(lower)
	switch {
	case strings.HasPrefix(lower, "meta-inf/") && (ext == ".rsa" || ext == ".dsa" || ext == ".ec"):
		return true
	case base == "embedded.mobileprovision", base == "androidmanifest.xml", base == "resources.arsc":
		return true
	case ext == ".dex", ext == ".so", ext == ".wasm", ext == ".dll", ext == ".dylib", ext == ".exe":
		return true
	case ext == "" && strings.Contains(lower, ".app/") && strings.Count(lower, "/") == 2:
		// The executable of an iOS app is the file named after the app.
		return true
	}
	return worthOpening(name)
}

// centralDirectory finds where a zip's central directory starts, which is
// where an APK's signing block ends.
func centralDirectory(data []byte) int {
	for i := len(data) - 22; i >= 0 && i >= len(data)-22-65535; i-- {
		if data[i] == 'P' && data[i+1] == 'K' && data[i+2] == 5 && data[i+3] == 6 {
			return int(uint32(data[i+16]) | uint32(data[i+17])<<8 | uint32(data[i+18])<<16 | uint32(data[i+19])<<24)
		}
	}
	return -1
}

// tarEntries reads an npm package or any other tarball, gzipped or not.
func tarEntries(data []byte) ([]entry, []string, error) {
	var r io.Reader = bytes.NewReader(data)
	if bytes.HasPrefix(data, []byte{0x1f, 0x8b}) {
		z, err := gzip.NewReader(r)
		if err != nil {
			return nil, nil, err
		}
		inner, err := io.ReadAll(io.LimitReader(z, 1<<30))
		if err != nil {
			return nil, nil, err
		}
		if !looksTar(inner) {
			// A file that was only gzipped: one entry, the file itself.
			return []entry{{"(inside the gzip)", inner}}, nil, nil
		}
		r = bytes.NewReader(inner)
	}
	tr := tar.NewReader(r)
	var out []entry
	var gaps []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, append(gaps, "the archive ends early"), nil
		}
		if h.Typeflag != tar.TypeReg || !worthInside(h.Name) {
			continue
		}
		if h.Size > maxEntry {
			gaps = append(gaps, fmt.Sprintf("%s is %s, more than this reads inside a container", h.Name, size(h.Size)))
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			gaps = append(gaps, h.Name+" could not be read")
			continue
		}
		out = append(out, entry{h.Name, body})
	}
	return out, gaps, nil
}

func looksTar(b []byte) bool {
	return len(b) > 262 && (string(b[257:262]) == "ustar")
}

// shower writes paths the way the report shows them: the home folder as
// ~, which keeps the one part of a path that names a person out of a
// report that might get pasted somewhere. --reveal shows it.
type shower struct {
	home   string
	reveal bool
}

func (s shower) path(p string) string {
	if s.reveal || s.home == "" {
		return p
	}
	rel, err := filepath.Rel(s.home, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return p
	}
	return "~/" + filepath.ToSlash(rel)
}

// text is a name that came out of a file, masked to the first and last
// letters of each word unless --reveal is given: enough for the owner to
// know their own, not enough for anybody else to read.
func (s shower) text(t string) string {
	t = strings.Join(strings.Fields(t), " ")
	if runes(t) > 80 {
		t = string([]rune(t)[:77]) + "..."
	}
	if s.reveal {
		return t
	}
	return maskText(t)
}

// id is an identifier, masked the same way but keeping its shape, so
// "UA-4815162-3" comes out as "UA-4*****2-3". A long token shows only the
// prefix that says what kind it is: half of a token is still somebody's.
func (s shower) id(v string) string {
	if s.reveal {
		if runes(v) > 100 {
			return string([]rune(v)[:97]) + "..."
		}
		return v
	}
	// A fingerprint keeps its two ends, which is how people compare them
	// by eye, and not enough to look one up.
	for _, p := range []string{"SHA-256 ", "SHA256:"} {
		if rest, ok := strings.CutPrefix(v, p); ok && len(rest) > 12 {
			return p + rest[:4] + "..." + rest[len(rest)-4:]
		}
	}
	if len(v) > 32 {
		if i := strings.LastIndexAny(v[:min(len(v), 12)], "._"); i > 0 {
			return v[:i+1] + "***"
		}
		return v[:6] + "***"
	}
	return maskID(v)
}
