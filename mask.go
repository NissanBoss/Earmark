package main

// Masking, the way Accent does it: first and last letter of every word,
// punctuation kept, so a name still reads as the shape of a name.

import "strings"

func maskWord(s string) string {
	r := []rune(s)
	switch {
	case len(r) == 0:
		return ""
	case len(r) <= 2:
		return string(r[0]) + strings.Repeat("*", len(r)-1)
	}
	return string(r[0]) + strings.Repeat("*", len(r)-2) + string(r[len(r)-1])
}

func maskText(s string) string {
	return maskRuns(s, 1, isWordRune)
}

// maskID leaves runs of three letters or fewer alone, which in an
// identifier are its prefix and its separators' company: UA-, pub-, the
// -2 at the end. The numbers between are what is masked, and so is every
// longer word but the names of the services themselves, which say what
// kind of address it is and nothing about whose.
func maskID(s string) string {
	return maskRuns(s, 4, func(r rune) bool { return r != '_' && isWordRune(r) })
}

var serviceWords = map[string]bool{
	"jsdelivr": true, "githubusercontent": true, "github": true, "gitlab": true, "githack": true, "rawcdn": true,
	"gist": true, "paypal": true, "buymeacoffee": true, "patreon": true, "liberapay": true, "opencollective": true,
	"workers": true, "calendly": true, "zendesk": true, "freshdesk": true, "typeform": true, "discord": true,
	"discordapp": true, "invite": true, "sponsors": true, "docker": true, "ghcr": true, "quay": true,
	"registry": true, "amazon": true, "cloudinary": true, "disqus": true, "auth0": true, "netlify": true,
	"vercel": true, "repl": true, "herokuapp": true, "onrender": true, "pages": true, "glitch": true,
	"surge": true, "azurewebsites": true, "amazonaws": true, "storage": true, "googleapis": true, "blob": true,
	"core": true, "windows": true, "supabase": true, "firebaseapp": true, "appspot": true, "firebaseio": true,
	"firebasestorage": true, "list": true, "manage": true, "gravatar": true, "avatar": true, "embed": true,
	"count": true, "stripe": true, "team": true, "live": true, "test": true, "https": true, "http": true,
}

// labelWords are this program's own, put in front of what a certificate
// or a profile says: "name", "city". The first release masked them along
// with the rest, and a certificate came out as n**e and c*****y.
var labelWords = map[string]bool{
	"name": true, "organisation": true, "unit": true, "city": true, "region": true,
	"country": true, "team": true, "certificate": true,
}

func maskRuns(s string, from int, inWord func(rune) bool) string {
	var b strings.Builder
	var word []rune
	flush := func() {
		if len(word) > 0 {
			lower := strings.ToLower(string(word))
			if len(word) >= from && !(from > 1 && serviceWords[lower]) && !labelWords[lower] {
				b.WriteString(maskWord(string(word)))
			} else {
				b.WriteString(string(word))
			}
			word = word[:0]
		}
	}
	for _, r := range s {
		if inWord(r) {
			word = append(word, r)
			continue
		}
		flush()
		b.WriteRune(r)
	}
	flush()
	return b.String()
}

func isWordRune(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r > 0x7f
}
