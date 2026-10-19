package main

// The identifiers this knows, what each one says about whoever put it
// there, and who can use it to find the rest of what they made.
//
// None of these is a secret, which is why nothing looks for them. A
// secret scanner skips a Mapbox public token on purpose, because anybody
// may see it; what anybody may also see is the user name written inside
// it. An AdSense publisher ID has to be on the page for the ads to work,
// and it is the same on every site of the account, which is how
// Bellingcat found networks of sites run by one hand. The question here
// is not whether a key can be abused but whether it is yours.

import (
	"path/filepath"
	"regexp"
	"strings"
)

type level int

const (
	// ties is an identifier of an account that only matches: whoever has
	// two projects of yours sees the same Sentry organisation in both,
	// and nobody else learns anything from one alone.
	ties level = iota
	// listed is one the reverse-lookup services index: give SpyOnWeb an
	// AdSense number and it lists every site that has ever carried it.
	listed
	// names is one that says who you are, or what you called an account
	// of yours: a user name, a team, a sign-in domain, an address.
	names
)

type mark struct {
	key   string // what it is called in tests, and the family of its ties
	name  string // what the report calls it
	level level
	hints []string // a literal, one of which must be in the file before the pattern is tried
	re    *regexp.Regexp
	read  func(m []string) (found, bool)
	// only limits a mark to some files. A github.com link in a minified
	// bundle is nearly always a library's own; in an HTML page or a
	// README it is somebody's.
	only   func(name string) bool
	lookup string
	advice string
}

type found struct {
	value string   // the identifier as it appears
	who   string   // the account name it gives away, if it gives one
	ties  []string // what it is matched on across projects
	level level
	note  string
}

func (m *mark) applies(name string) bool {
	return m.only == nil || m.only(name)
}

// one reads the first group that matched, for patterns written as
// alternatives, and ties it to the mark's family.
func one(family string, lv level, who bool) func(m []string) (found, bool) {
	return func(m []string) (found, bool) {
		v := firstGroup(m)
		if v == "" || placeholder(v) {
			return found{}, false
		}
		f := found{value: v, ties: []string{family + ":" + strings.ToLower(v)}, level: lv}
		if who {
			f.who = v
		}
		return f, true
	}
}

func firstGroup(m []string) string {
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return ""
}

// placeholder is what documentation and templates put where the real
// thing goes. Reporting UA-XXXXX-Y as your Analytics account is the
// quickest way to make a report nobody reads twice.
func placeholder(v string) bool {
	l := strings.ToLower(v)
	for _, w := range []string{"xxxx", "your", "example", "sample", "placeholder", "dummy", "changeme", "replace", "insert", "1234567", "0000000", "abcdef", "abc123", "test123", "foobar", "lorem", "my-app", "myapp", "username", "yourname", "projectid", "project-id"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	alnum := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r >= 'a' && r <= 'z' {
			return r
		}
		return -1
	}, l)
	if len(alnum) >= 4 && strings.Count(alnum, alnum[:1]) == len(alnum) {
		return true
	}
	return false
}

// Accounts that documentation uses, which are not anybody's project.
var docsAccounts = map[string]bool{
	"examples": true, "mapbox": true, "demo": true, "octocat": true, "user": true, "me": true,
	"example": true, "test": true, "foo": true, "bar": true, "acme": true, "john": true, "johndoe": true,
	"jane": true, "janedoe": true, "someone": true, "name": true, "account": true, "owner": true,
}

var markup = map[string]bool{
	".html": true, ".htm": true, ".md": true, ".markdown": true, ".mdx": true, ".txt": true,
	".json": true, ".yml": true, ".yaml": true, ".toml": true, ".xml": true, ".plist": true,
	".webmanifest": true, ".cfg": true, ".ini": true, ".rst": true, ".adoc": true,
}

func markupOnly(name string) bool {
	return markup[strings.ToLower(filepath.Ext(name))] || filepath.Ext(name) == ""
}

func named(names ...string) func(string) bool {
	return func(name string) bool {
		base := strings.ToLower(filepath.Base(name))
		for _, n := range names {
			if base == n {
				return true
			}
		}
		return false
	}
}

// githubReserved are the first parts of github.com addresses that are
// pages of GitHub's, not accounts.
var githubReserved = map[string]bool{
	"about": true, "apps": true, "collections": true, "contact": true, "customer-stories": true,
	"enterprise": true, "events": true, "explore": true, "features": true, "github": true,
	"issues": true, "join": true, "login": true, "logout": true, "marketplace": true, "new": true,
	"notifications": true, "orgs": true, "organizations": true, "pricing": true, "pulls": true,
	"readme": true, "search": true, "security": true, "settings": true, "site": true,
	"sponsors": true, "topics": true, "trending": true, "users": true, "codespaces": true,
	"copilot": true, "solutions": true, "team": true, "account": true, "dashboard": true, "advisories": true,
}

// Paths after a donation site's address that are its own pages.
var siteReserved = map[string]bool{
	"join": true, "login": true, "signup": true, "home": true, "posts": true, "c": true, "u": true,
	"explore": true, "search": true, "about": true, "pricing": true, "privacy": true, "terms": true,
	"discover": true, "apps": true, "help": true, "manage": true, "legal": true, "s": true, "share": true,
	"joinchat": true, "addstickers": true, "iv": true, "proxy": true, "contribute": true,
}

// commonAccounts are linked from so many unrelated projects that a link
// to one says which theme or library a site uses, not whose it is. In the
// first forty personal sites tried, a third of the GitHub accounts found
// were these: the theme's author in the footer, the library's in a
// credit. Two sites on the same theme are not one person's.
var commonAccounts = map[string]bool{
	"github:academicpages": true, "github:alshedivat": true, "github:mmistakes": true, "github:jekyll": true,
	"github:twbs": true, "github:fortawesome": true, "github:microsoft": true, "github:google": true,
	"github:apache": true, "github:facebook": true, "github:vercel": true, "github:gohugoio": true,
	"github:hexojs": true, "github:mkdocs": true, "github:squidfunk": true, "github:pages-themes": true,
	"github:googlecloudplatform": true, "github:pytest-dev": true, "github:quarto-dev": true,
	"github:mermaid-js": true, "github:swiftlang": true, "github:zulip": true, "github:jquery": true,
	"github:mathjax": true, "github:highlightjs": true, "github:primer": true, "github:github": true,
	"github:npm": true, "github:nodejs": true, "github:python": true, "github:rust-lang": true,
	"github:golang": true, "github:docsifyjs": true, "github:vuejs": true, "github:sveltejs": true,
	"github:withastro": true, "github:tailwindlabs": true, "github:alpinejs": true, "github:reactjs": true,
	"github:angular": true, "github:jgthms": true, "github:chartjs": true, "github:d3": true,
	"github:leaflet": true, "github:mapbox": true, "github:cdnjs": true, "github:jsdelivr": true,
}

func account(family string, reserved map[string]bool) func(m []string) (found, bool) {
	return func(m []string) (found, bool) {
		v := firstGroup(m)
		l := strings.ToLower(v)
		if v == "" || placeholder(v) || docsAccounts[l] || reserved[l] || commonAccounts[family+":"+l] {
			return found{}, false
		}
		// The address it was found in, where that reads as one, so the
		// report says ko-fi.com/name and not just the name.
		shown := v
		if !strings.ContainsAny(m[0], "\"'<> \t\n") && len(m[0]) <= 80 {
			shown = strings.TrimPrefix(strings.TrimPrefix(m[0], "https://"), "http://")
		}
		return found{value: shown, who: v, ties: []string{family + ":" + l}, level: names}, true
	}
}

var (
	lookupInside   = "Written in the identifier itself, for anybody who opens the page source."
	lookupIndexes  = "SpyOnWeb, SameID, DNSlytics and BuiltWith list every site that carries it, and keep listing sites that dropped it years ago."
	lookupMatch    = "Only the company can say whose it is, but it is the same in everything the account makes, so anything else of yours that carries it is tied to this."
	lookupAddress  = "It is in the address: anybody who reads it reads the account."
	adviceSeparate = "A project that must stand apart from you needs an account of its own with this provider. Hiding or restricting the identifier does not help: the page needs it to work, and what matters is whose it is."
	adviceIndexes  = "A project that must stand apart from you needs an account of its own. Taking the identifier out now does not take it out of the indexes, which remember what a site carried."
	adviceAddress  = "Host it, or link to it, from an account that is the project's own. An address on somebody's account names them to everybody who looks."
)

var marks = []*mark{
	{
		key: "mapbox", name: "Mapbox token", level: names,
		hints: []string{"pk.eyJ", "sk.eyJ"},
		re:    regexp.MustCompile(`\b([ps]k)\.(eyJ[0-9A-Za-z_-]{8,})\.[0-9A-Za-z_-]{8,}`),
		read: func(m []string) (found, bool) {
			u, ok := mapboxUser(m[2])
			if !ok || docsAccounts[strings.ToLower(u)] {
				return found{}, false
			}
			f := found{value: m[0], who: u, ties: []string{"mapbox:" + strings.ToLower(u)}, level: names}
			if m[1] == "sk" {
				f.note = "This one is a secret token, not a public one: anybody holding it can act as the account, so it wants replacing whatever else is done."
			}
			return f, true
		},
		lookup: "The user name is written inside the token, as base64 of a line of JSON, for anybody who opens the page source.",
		advice: "A Mapbox token always names the account that made it. A project that must stand apart from you needs a Mapbox account of its own; restricting the token to a domain stops other sites using it, not anybody reading it.",
	},
	{
		key: "clerk", name: "Clerk publishable key", level: names,
		hints: []string{"pk_live_", "pk_test_"},
		re:    regexp.MustCompile(`\bpk_(live|test)_([A-Za-z0-9+/_-]{12,}={0,2})`),
		read: func(m []string) (found, bool) {
			host, ok := clerkHost(m[2])
			if !ok {
				return found{}, false
			}
			f := found{value: m[0], ties: []string{"clerk:" + host}, level: ties}
			if m[1] == "live" {
				// A production instance signs people in on your own domain,
				// and the key is that domain spelled out.
				f.who, f.level = host, names
			}
			return f, true
		},
		lookup: "The key is the sign-in address of your Clerk instance, written in base64. A production key spells out your own domain.",
		advice: adviceSeparate,
	},
	{
		key: "stripe", name: "Stripe publishable key", level: ties,
		hints: []string{"pk_live_", "pk_test_"},
		re:    regexp.MustCompile(`\bpk_(?:live|test)_([0-9A-Za-z]{24,107})\b`),
		read: func(m []string) (found, bool) {
			if _, clerk := clerkHost(m[1]); clerk || placeholder(m[1]) {
				return found{}, false
			}
			return found{value: m[0], ties: []string{"stripe:" + m[0]}, level: ties}, true
		},
		lookup: lookupMatch,
		advice: "Stripe allows several accounts to one person; a project that must stand apart needs its own, or every page that takes a payment carries the same key.",
	},
	{
		key: "stripe-link", name: "Stripe payment link", level: names,
		hints:  []string{"buy.stripe.com"},
		re:     regexp.MustCompile(`buy\.stripe\.com/((?:test_)?[0-9A-Za-z]{12,})`),
		read:   one("stripe-link", names, false),
		lookup: "The page it opens shows the business name of the account, and so does the line on the buyer's card statement.",
		advice: adviceSeparate,
	},
	{
		key: "ga", name: "Google Analytics account", level: listed,
		hints: []string{"UA-"},
		re:    regexp.MustCompile(`\bUA-(\d{4,10})-(\d{1,4})\b`),
		read: func(m []string) (found, bool) {
			if placeholder(m[1]) {
				return found{}, false
			}
			// The number before the last dash is the account, and every
			// property of the account shares it.
			return found{value: m[0], ties: []string{"ga:" + m[1]}, level: listed}, true
		},
		lookup: lookupIndexes,
		advice: "The number in the middle is the account, the same for every property in it. " + adviceIndexes,
	},
	{
		key: "ga4", name: "Google Analytics measurement ID", level: listed,
		hints:  []string{"G-"},
		re:     regexp.MustCompile(`(?:gtag\(\s*['"]config['"]\s*,\s*['"]|googletagmanager\.com/gtag/js\?id=|[mM]easurement_?[iI][dD]['"]?\s*[:=]\s*['"])(G-[A-Z0-9]{6,12})\b`),
		read:   one("ga4", listed, false),
		lookup: lookupIndexes,
		advice: adviceIndexes,
	},
	{
		key: "gtm", name: "Google Tag Manager container", level: listed,
		hints:  []string{"GTM-"},
		re:     regexp.MustCompile(`\b(GTM-[A-Z0-9]{4,9})\b`),
		read:   one("gtm", listed, false),
		lookup: lookupIndexes,
		advice: adviceIndexes,
	},
	{
		key: "google-ads", name: "Google Ads tag", level: listed,
		hints:  []string{"AW-"},
		re:     regexp.MustCompile(`\b(AW-\d{9,11})\b`),
		read:   one("google-ads", listed, false),
		lookup: lookupIndexes,
		advice: adviceIndexes,
	},
	{
		key: "adsense", name: "Google AdSense publisher ID", level: listed,
		hints: []string{"pub-"},
		// Not preceded by "app-": that is AdMob's form of the same number,
		// read below.
		re: regexp.MustCompile(`(?:^|[^-a-z0-9])(?:ca-)?pub-(\d{16})\b`),
		read: func(m []string) (found, bool) {
			if placeholder(m[1]) {
				return found{}, false
			}
			return found{value: "pub-" + m[1], ties: []string{"google-pub:" + m[1]}, level: listed}, true
		},
		lookup: lookupIndexes + " Bellingcat has followed AdSense numbers to whole networks of sites.",
		advice: "Google allows one AdSense account to a person, so this number is you on every site that shows your ads, and on every app if you use AdMob, which shows the same number. A site that must stand apart from you cannot carry your ads.",
	},
	{
		key: "admob", name: "Google AdMob app ID", level: listed,
		hints: []string{"ca-app-pub-"},
		re:    regexp.MustCompile(`\bca-app-pub-(\d{16})(?:[~/]\d{6,12})?`),
		read: func(m []string) (found, bool) {
			if placeholder(m[1]) {
				return found{}, false
			}
			return found{value: m[0], ties: []string{"google-pub:" + m[1]}, level: listed}, true
		},
		lookup: "The sixteen digits are the publisher, the same number as on any website of the account that shows AdSense ads. " + lookupIndexes,
		advice: "Google allows one AdMob and AdSense publisher to a person, and app stores show every app that carries it. An app that must stand apart from you cannot carry your ads.",
	},
	{
		key: "google-key", name: "Google API key", level: ties,
		hints:  []string{"AIza"},
		re:     regexp.MustCompile(`\b(AIza[0-9A-Za-z_-]{35})(?:[^0-9A-Za-z_-]|$)`),
		read:   one("google-key", ties, false),
		lookup: "Google can say which Cloud project it belongs to; anybody else can match it against the same key in another project.",
		advice: "A key belongs to one Cloud project. A project that must stand apart from you needs a Cloud project of its own; restricting the key limits what others can do with it, not what it says.",
	},
	{
		key: "google-project", name: "Google Cloud project number", level: ties,
		hints: []string{"apps.googleusercontent.com", ":web:", ":android:", ":ios:", "messagingSenderId", "project_number"},
		re: regexp.MustCompile(`\b(\d{6,14})-[0-9a-z]{32}\.apps\.googleusercontent\.com|\b1:(\d{6,14}):(?:web|android|ios):[0-9a-f]{6,}` +
			`|messagingSenderId['"]?\s*[:=]\s*['"](\d{6,14})['"]|"project_number"\s*:\s*"(\d{6,14})"`),
		read: one("google-project", ties, false),
		lookup: "The same number is in every sign-in client, every Firebase app and every notification sender the Cloud project has, " +
			"so it ties together whatever was made in one project.",
		advice: "A project that must stand apart from you needs a Cloud project of its own, made from an account that is not yours.",
	},
	{
		key: "firebase", name: "Firebase project", level: ties,
		hints: []string{"firebaseapp.com", "appspot.com", "firebasestorage.app", "firebaseio.com", "project_id", ".firebaserc"},
		re: regexp.MustCompile(`\b([a-z0-9][a-z0-9-]{4,38}[a-z0-9])(?:-default-rtdb)?\.(?:firebaseapp\.com|appspot\.com|firebasestorage\.app|firebaseio\.com)\b` +
			`|"project_id"\s*:\s*"([a-z0-9][a-z0-9-]{4,28}[a-z0-9])"`),
		read: func(m []string) (found, bool) {
			v := firstGroup(m)
			if v == "" || placeholder(v) {
				return found{}, false
			}
			return found{value: v, who: v, ties: []string{"firebase:" + v}, level: ties}, true
		},
		lookup: "The project ID is a name somebody chose, in every address the project serves from.",
		advice: "A project that must stand apart from you needs a Firebase project of its own, made from an account that is not yours; a name that says nothing helps only with the name.",
	},
	{
		key: "sentry", name: "Sentry DSN", level: ties,
		hints: []string{"sentry.io"},
		re:    regexp.MustCompile(`https?://[0-9a-f]{32}(?::[0-9a-f]{32})?@(o\d+)\.ingest(?:\.[a-z]{2})?\.sentry\.io/(\d+)|https?://[0-9a-f]{32}(?::[0-9a-f]{32})?@sentry\.io/(\d+)`),
		read: func(m []string) (found, bool) {
			// The key in front of the @ is the project's to send with; what
			// ties it is the organisation, so that is what is shown.
			if m[1] != "" {
				return found{value: m[1] + "/" + m[2], ties: []string{"sentry:" + m[1]}, level: ties}, true
			}
			return found{value: "project " + m[3], ties: []string{"sentry-project:" + m[3]}, level: ties}, true
		},
		lookup: "The o and its number are the Sentry organisation, the same in every project of it. " + lookupMatch,
		advice: adviceSeparate,
	},
	{
		key: "supabase", name: "Supabase project", level: ties,
		hints:  []string{"supabase.co"},
		re:     regexp.MustCompile(`\b([a-z]{20})\.supabase\.co\b`),
		read:   one("supabase", ties, false),
		lookup: lookupMatch,
		advice: adviceSeparate,
	},
	{
		key: "jwt", name: "token", level: names,
		hints: []string{"eyJ"},
		re:    regexp.MustCompile(`\beyJ[0-9A-Za-z_-]{8,}\.eyJ[0-9A-Za-z_-]{8,}\.[0-9A-Za-z_-]{8,}`),
		read: func(m []string) (found, bool) {
			claims := jwtClaims(m[0])
			if claims == nil {
				return found{}, false
			}
			if claims["iss"] == "supabase" {
				ref, _ := claims["ref"].(string)
				if ref == "" {
					return found{}, false
				}
				f := found{value: m[0], ties: []string{"supabase:" + ref}, level: ties}
				if claims["role"] == "service_role" {
					f.note = "This is the service key, which gets past every rule the database has; it wants replacing whatever else is done."
				}
				return f, true
			}
			// A token copied from a real session into a test or an example
			// says who was signed in.
			for _, k := range []string{"email", "preferred_username", "upn", "unique_name", "name", "nickname"} {
				if v, _ := claims[k].(string); v != "" && !placeholder(v) && !docsAccounts[strings.ToLower(v)] {
					return found{value: m[0], who: v, ties: []string{"person:" + strings.ToLower(v)}, level: names}, true
				}
			}
			return found{}, false
		},
		lookup: "A token is signed, not hidden: the middle part is base64 of what it says, for anybody to read.",
		advice: "A token from a real session does not belong in a project at all. Make one for the test with a name that is nobody's.",
	},
	{
		key: "auth0", name: "Auth0 tenant", level: names,
		hints:  []string{"auth0.com"},
		re:     regexp.MustCompile(`\b([a-z0-9][a-z0-9-]{1,60})(?:\.[a-z]{2})?\.auth0\.com\b`),
		read:   account("auth0", map[string]bool{"cdn": true, "www": true, "login": true, "manage": true, "community": true, "marketplace": true, "support": true}),
		lookup: lookupAddress,
		advice: adviceSeparate,
	},
	{
		key: "algolia", name: "Algolia application", level: ties,
		hints:  []string{"algolia"},
		re:     regexp.MustCompile(`\b([A-Z0-9]{10})-dsn\.algolia\.net|\b([a-z0-9]{10})-dsn\.algolia\.net|(?i:algolia)[\s\S]{0,200}?(?:application_?id|app_?id|appId)['"]?\s*[:=]\s*['"]([A-Z0-9]{10})['"]`),
		read:   one("algolia", ties, false),
		lookup: lookupMatch,
		advice: adviceSeparate,
	},
	{
		key: "cloudinary", name: "Cloudinary account", level: ties,
		hints: []string{"cloudinary.com"},
		re:    regexp.MustCompile(`res\.cloudinary\.com/([a-z0-9_-]{3,40})/`),
		read: func(m []string) (found, bool) {
			if placeholder(m[1]) || docsAccounts[m[1]] {
				return found{}, false
			}
			return found{value: m[1], who: m[1], ties: []string{"cloudinary:" + m[1]}, level: ties}, true
		},
		lookup: "The cloud name is in the address of every picture the account serves. Older accounts chose it; newer ones were given one.",
		advice: adviceSeparate,
	},
	{
		key: "mailchimp", name: "Mailchimp sign-up form", level: names,
		hints: []string{"list-manage.com"},
		re:    regexp.MustCompile(`\b([a-z0-9][a-z0-9-]{1,60})\.us\d{1,2}\.list-manage\.com/subscribe/post\?u=([0-9a-f]{20,32})`),
		read: func(m []string) (found, bool) {
			return found{value: m[0], who: m[1], ties: []string{"mailchimp:" + m[2], "mailchimp-name:" + m[1]}, level: names}, true
		},
		lookup: "The address starts with the account's own name, and the u= after it is the account: every form it makes carries both.",
		advice: adviceSeparate,
	},
	{
		key: "amazon", name: "Amazon Associates tag", level: names,
		hints:  []string{"tag="},
		re:     regexp.MustCompile(`amazon\.[a-z.]{2,6}/[^"'\s<>]*?[?&](?:amp;)?tag=([a-z0-9][a-z0-9-]{1,40}-2[0-9])\b`),
		read:   account("amazon", nil),
		lookup: "The tag is a name the account chose, and the same tag on two sites is one account's commission.",
		advice: adviceSeparate,
	},
	{
		key: "gravatar", name: "Gravatar picture", level: names,
		hints: []string{"gravatar.com/avatar/"},
		re:    regexp.MustCompile(`gravatar\.com/avatar/([0-9a-f]{64}|[0-9a-f]{32})\b`),
		read: func(m []string) (found, bool) {
			// The hash of an empty address, which themes use for "nobody".
			if m[1] == "d41d8cd98f00b204e9800998ecf8427e" || m[1] == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" || placeholder(m[1]) {
				return found{}, false
			}
			return found{value: m[1], ties: []string{"gravatar:" + m[1]}, level: names,
				note: "It is the hash of an email address. Anybody who has the address can check it, and Gravatar answers with the profile for it."}, true
		},
		lookup: "Gravatar turns the hash into a profile, and anybody with a guess at the address can confirm it.",
		advice: "Use an address for the project that is not yours anywhere else, or no picture.",
	},
	{
		key: "disqus", name: "Disqus site", level: names,
		hints:  []string{"disqus.com"},
		re:     regexp.MustCompile(`\b([a-z0-9][a-z0-9-]{1,60})\.disqus\.com/(?:embed|count)\.js`),
		read:   account("disqus", nil),
		lookup: lookupAddress,
		advice: adviceSeparate,
	},
	{
		key: "facebook-pixel", name: "Meta Pixel", level: listed,
		hints:  []string{"fbq("},
		re:     regexp.MustCompile(`fbq\(\s*['"]init['"]\s*,\s*['"](\d{12,17})['"]`),
		read:   one("facebook-pixel", listed, false),
		lookup: lookupIndexes,
		advice: adviceIndexes,
	},
	{
		key: "facebook-app", name: "Facebook app", level: ties,
		hints:  []string{"fb:app_id"},
		re:     regexp.MustCompile(`fb:app_id["']\s+content=["'](\d{12,17})["']|content=["'](\d{12,17})["']\s+property=["']fb:app_id["']`),
		read:   one("facebook-app", ties, false),
		lookup: "Meta shows the name of the app, and the app's page shows who runs it.",
		advice: adviceSeparate,
	},
	{
		key: "facebook-admins", name: "Facebook admins tag", level: names,
		hints:  []string{"fb:admins"},
		re:     regexp.MustCompile(`fb:admins["']\s+content=["']([0-9, ]{5,})["']|content=["']([0-9, ]{5,})["']\s+property=["']fb:admins["']`),
		read:   one("facebook-admins", names, true),
		lookup: "These are Facebook user IDs, and facebook.com/ followed by one opens that person's profile.",
		advice: "The tag is not needed by anything current; take it out.",
	},
	{
		key: "twitter", name: "handle on X", level: names,
		hints:  []string{"twitter:"},
		re:     regexp.MustCompile(`twitter:(?:creator|site)["']\s+content=["']@?([A-Za-z0-9_]{1,15})["']|content=["']@?([A-Za-z0-9_]{1,15})["']\s+(?:name|property)=["']twitter:(?:creator|site)["']`),
		read:   account("twitter", nil),
		lookup: "It is a handle, shown as is to anybody who shares the page.",
		advice: "Themes and templates fill these in with the author's handle. Take it out, or put the project's own.",
	},
	{
		key: "hotjar", name: "Hotjar site", level: ties,
		hints:  []string{"hjid"},
		re:     regexp.MustCompile(`hjid\s*:\s*(\d{5,9})`),
		read:   one("hotjar", ties, false),
		lookup: lookupIndexes,
		advice: adviceIndexes,
	},
	{
		key: "clarity", name: "Microsoft Clarity project", level: ties,
		hints:  []string{"clarity"},
		re:     regexp.MustCompile(`clarity\.ms/tag/([a-z0-9]{8,12})|["']clarity["']\s*,\s*["']script["']\s*,\s*["']([a-z0-9]{8,12})["']`),
		read:   one("clarity", ties, false),
		lookup: lookupMatch,
		advice: adviceSeparate,
	},
	{
		key: "tiktok", name: "TikTok pixel", level: listed,
		hints:  []string{"ttq.load"},
		re:     regexp.MustCompile(`ttq\.load\(\s*['"]([A-Z0-9]{18,22})['"]`),
		read:   one("tiktok", listed, false),
		lookup: lookupIndexes,
		advice: adviceIndexes,
	},
	{
		key: "linkedin", name: "LinkedIn Insight tag", level: listed,
		hints:  []string{"_linkedin_partner_id"},
		re:     regexp.MustCompile(`_linkedin_partner_id\s*=\s*["'](\d{4,10})["']`),
		read:   one("linkedin", listed, false),
		lookup: lookupIndexes,
		advice: adviceIndexes,
	},
	{
		key: "cloudflare-analytics", name: "Cloudflare Web Analytics", level: ties,
		hints:  []string{"cf-beacon", "cloudflareinsights"},
		re:     regexp.MustCompile(`data-cf-beacon=['"]\{\s*(?:&quot;|")token(?:&quot;|")\s*:\s*(?:&quot;|")([0-9a-f]{32})`),
		read:   one("cloudflare-analytics", ties, false),
		lookup: lookupMatch,
		advice: adviceSeparate,
	},
	{
		key: "cloudflare-account", name: "Cloudflare account", level: ties,
		hints:  []string{"account_id"},
		re:     regexp.MustCompile(`\baccount_id\s*=\s*["']([0-9a-f]{32})["']|"account_id"\s*:\s*"([0-9a-f]{32})"`),
		read:   one("cloudflare-account", ties, false),
		lookup: "It is the account, the same in every Worker and every Pages project it has. " + lookupMatch,
		advice: "Keep the account out of a public repository with an environment variable, CLOUDFLARE_ACCOUNT_ID, which Wrangler reads instead.",
	},
	{
		key: "workers", name: "Cloudflare Workers address", level: names,
		hints: []string{"workers.dev"},
		re:    regexp.MustCompile(`\b([a-z0-9-]{1,63})\.([a-z0-9-]{1,63})\.workers\.dev\b`),
		read: func(m []string) (found, bool) {
			sub := m[2]
			if placeholder(sub) || docsAccounts[sub] {
				return found{}, false
			}
			return found{value: m[0], who: sub, ties: []string{"workers:" + sub}, level: names}, true
		},
		lookup: "The part before workers.dev is the account's own subdomain, one per account, the same for every Worker it runs.",
		advice: adviceAddress,
	},
	{
		key: "github", name: "GitHub account", level: names,
		hints: []string{"github.io", "githubusercontent.com", "jsdelivr.net/gh/", "githack.com"},
		re: regexp.MustCompile(`\b([a-z0-9][a-z0-9-]{0,38})\.github\.io\b` +
			`|(?:raw\.githubusercontent\.com|gist\.githubusercontent\.com|cdn\.jsdelivr\.net/gh|raw\.githack\.com|rawcdn\.githack\.com)/([A-Za-z0-9][A-Za-z0-9-]{0,38})/`),
		read:   account("github", githubReserved),
		lookup: "The account is in the address, and every repository, star and commit of it is a click away.",
		advice: adviceAddress,
	},
	{
		key: "github-link", name: "GitHub account", level: names,
		hints:  []string{"github.com/"},
		re:     regexp.MustCompile(`\bgithub\.com/(?:sponsors/)?([A-Za-z0-9][A-Za-z0-9-]{0,38})\b`),
		read:   account("github", githubReserved),
		only:   markupOnly,
		lookup: "The account is in the address, and every repository, star and commit of it is a click away.",
		advice: adviceAddress,
	},
	{
		key: "gitlab", name: "GitLab account", level: names,
		hints:  []string{"gitlab.io"},
		re:     regexp.MustCompile(`\b([a-z0-9][a-z0-9-]{0,38})\.gitlab\.io\b`),
		read:   account("gitlab", nil),
		lookup: lookupAddress,
		advice: adviceAddress,
	},
	{
		key: "vercel", name: "Vercel deployment", level: ties,
		hints: []string{"vercel.app"},
		re:    regexp.MustCompile(`\b([a-z0-9][a-z0-9-]{1,62})\.vercel\.app\b`),
		read: func(m []string) (found, bool) {
			if placeholder(m[1]) {
				return found{}, false
			}
			if scope, ok := vercelScope(m[1]); ok && !placeholder(scope) {
				// A deployment's own address ends in the account or team it
				// belongs to; a personal account's is its name and -projects.
				return found{value: m[0], who: scope, ties: []string{"vercel-scope:" + scope}, level: names}, true
			}
			return found{value: m[0], ties: []string{"vercel:" + m[1]}, level: ties}, true
		},
		lookup: "A deployment address ends with the account or team it was deployed from.",
		advice: "Link to the production address, the project's own name and nothing after it, and deploy from an account that is the project's.",
	},
	{
		key: "netlify", name: "Netlify site", level: ties,
		hints:  []string{"netlify.app"},
		re:     regexp.MustCompile(`\b(?:deploy-preview-\d+--|[0-9a-f]{24}--)?([a-z0-9][a-z0-9-]{1,62})\.netlify\.app\b`),
		read:   one("netlify", ties, false),
		lookup: lookupAddress,
		advice: adviceAddress,
	},
	{
		key: "replit", name: "Replit account", level: names,
		hints: []string{"repl.co"},
		re:    regexp.MustCompile(`\b[a-z0-9-]+\.([a-z0-9-]+)\.repl\.co\b`),
		read:  account("replit", nil), lookup: lookupAddress, advice: adviceAddress,
	},
	{
		key: "app-host", name: "app address", level: ties,
		hints: []string{"herokuapp.com", "fly.dev", "onrender.com", "pages.dev", ".web.app", "glitch.me", "surge.sh", "azurewebsites.net"},
		re:    regexp.MustCompile(`\b([a-z0-9][a-z0-9-]{1,62})\.(herokuapp\.com|fly\.dev|onrender\.com|pages\.dev|web\.app|glitch\.me|surge\.sh|azurewebsites\.net)\b`),
		read: func(m []string) (found, bool) {
			if placeholder(m[1]) || docsAccounts[m[1]] || m[1] == "www" {
				return found{}, false
			}
			return found{value: m[1] + "." + m[2], ties: []string{"host:" + m[1] + "." + m[2]}, level: ties}, true
		},
		lookup: lookupAddress,
		advice: adviceAddress,
	},
	{
		key: "bucket", name: "storage bucket", level: ties,
		hints: []string{"amazonaws.com", "storage.googleapis.com", "blob.core.windows.net"},
		re: regexp.MustCompile(`\b([a-z0-9][a-z0-9.-]{1,61}[a-z0-9])\.s3[.-](?:[a-z0-9-]+\.)?amazonaws\.com\b` +
			`|\bs3[.-](?:[a-z0-9-]+\.)?amazonaws\.com/([a-z0-9][a-z0-9.-]{1,61}[a-z0-9])/` +
			`|storage\.googleapis\.com/([a-z0-9][a-z0-9._-]{1,61}[a-z0-9])/` +
			`|\b([a-z0-9]{3,24})\.blob\.core\.windows\.net\b`),
		read: func(m []string) (found, bool) {
			v := firstGroup(m)
			if v == "" || placeholder(v) || v == "s3" {
				return found{}, false
			}
			return found{value: v, who: v, ties: []string{"bucket:" + v}, level: ties}, true
		},
		lookup: "A bucket's name is one somebody chose, unique across the whole provider, and it is in every address it serves.",
		advice: adviceAddress,
	},
	{
		key: "registry", name: "container image account", level: names,
		hints:  []string{"ghcr.io/", "docker.io/", "quay.io/", "registry.gitlab.com/"},
		re:     regexp.MustCompile(`\b(?:ghcr\.io|docker\.io|quay\.io|registry\.gitlab\.com)/([a-z0-9][a-z0-9_.-]{0,38})/[a-z0-9][a-z0-9_.-]*`),
		read:   account("registry", map[string]bool{"library": true}),
		lookup: lookupAddress,
		advice: adviceAddress,
	},
	{
		key: "donation", name: "payment page", level: names,
		hints:  []string{"paypal.me/", "ko-fi.com/", "buymeacoffee.com/", "patreon.com/", "liberapay.com/", "opencollective.com/"},
		re:     regexp.MustCompile(`\b(?:paypal\.me|ko-fi\.com|buymeacoffee\.com|patreon\.com|liberapay\.com|opencollective\.com)/([A-Za-z0-9_.-]{2,40})`),
		read:   account("donation", siteReserved),
		lookup: lookupAddress,
		advice: "Money goes to an account, and the account is somebody's. A project that must stand apart from you needs a payment page of its own.",
	},
	{
		key: "paypal-email", name: "PayPal button address", level: names,
		hints:  []string{"business="},
		re:     regexp.MustCompile(`[?&](?:amp;)?business=([A-Za-z0-9._%+-]+(?:@|%40)[A-Za-z0-9.-]+\.[a-z]{2,})`),
		read:   account("email", nil),
		lookup: "The button sends the money to this address, which is written out in the page.",
		advice: "Use a hosted button, which carries an ID instead of the address, from an account that is the project's.",
	},
	{
		key: "email", name: "email address", level: names,
		hints: []string{"mailto:"},
		re:    regexp.MustCompile(`mailto:([A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,})`),
		read: func(m []string) (found, bool) {
			v := strings.ToLower(m[1])
			domain := v[strings.LastIndexByte(v, '@')+1:]
			if placeholder(v) || domain == "example.com" || domain == "example.org" || domain == "domain.com" || domain == "email.com" {
				return found{}, false
			}
			return found{value: m[1], who: m[1], ties: []string{"email:" + v}, level: names}, true
		},
		lookup: "It is an address, for anybody to write to and to search for.",
		advice: "Give the project an address of its own.",
	},
	{
		key: "phone", name: "WhatsApp number", level: names,
		hints:  []string{"wa.me/"},
		re:     regexp.MustCompile(`\bwa\.me/\+?(\d{7,15})`),
		read:   one("phone", names, true),
		lookup: "It is a phone number, and the link opens a chat with whoever has it.",
		advice: "A number is a person. Use a business account on a number of its own, or no link.",
	},
	{
		key: "telegram", name: "Telegram account", level: names,
		hints:  []string{"t.me/"},
		re:     regexp.MustCompile(`\bt\.me/([A-Za-z][A-Za-z0-9_]{4,31})\b`),
		read:   account("telegram", siteReserved),
		lookup: lookupAddress,
		advice: adviceAddress,
	},
	{
		key: "discord", name: "Discord invite", level: ties,
		hints:  []string{"discord.gg/", "discord.com/invite/", "discordapp.com/invite/"},
		re:     regexp.MustCompile(`\bdiscord(?:\.gg|(?:app)?\.com/invite)/([A-Za-z0-9-]{2,32})\b`),
		read:   one("discord", ties, false),
		lookup: "An invite opens the server, and the server shows who owns it.",
		advice: "Invite people to a server made by an account that is the project's.",
	},
	{
		key: "service", name: "support or booking page", level: names,
		hints: []string{"calendly.com/", "zendesk.com", "freshdesk.com", "typeform.com"},
		re:    regexp.MustCompile(`\bcalendly\.com/([A-Za-z0-9_-]{3,60})|\b([a-z0-9][a-z0-9-]{1,60})\.(?:zendesk|freshdesk|typeform)\.com\b`),
		read: account("service", map[string]bool{"form": true, "help": true, "support": true, "www": true, "api": true,
			"static": true, "embed": true, "admin": true, "assets": true, "app": true, "developer": true, "status": true}),
		lookup: lookupAddress,
		advice: adviceAddress,
	},
	{
		key: "chat-widget", name: "chat widget", level: ties,
		hints: []string{"intercom", "CRISP_WEBSITE_ID", "tawk.to", "hs-scripts.com"},
		re: regexp.MustCompile(`widget\.intercom\.io/widget/([a-z0-9]{8})|intercomSettings[\s\S]{0,200}?app_id['"]?\s*:\s*['"]([a-z0-9]{8})['"]` +
			`|CRISP_WEBSITE_ID\s*=\s*["']([0-9a-f-]{36})["']|embed\.tawk\.to/([0-9a-f]{24})/|js(?:-[a-z0-9]+)?\.hs-scripts\.com/(\d{5,10})\.js`),
		read:   one("chat-widget", ties, false),
		lookup: lookupMatch,
		advice: adviceSeparate,
	},
	{
		key: "analytics-host", name: "self-hosted analytics server", level: names,
		hints: []string{".js"},
		// Plausible, Umami, Matomo and Fathom can run on your own server,
		// and then the script comes from it.
		re: regexp.MustCompile(`<script\b[^>]*?src=["'](?:https?:)?//([a-z0-9][a-z0-9.-]*\.[a-z]{2,})/(?:js/)?(?:script|plausible|umami|matomo|piwik|tracker)[a-z0-9.-]*\.js`),
		read: func(m []string) (found, bool) {
			host := strings.ToLower(m[1])
			for _, vendor := range []string{"plausible.io", "umami.is", "usefathom.com", "matomo.cloud", "googletagmanager.com", "jsdelivr.net", "unpkg.com", "cloudflare.com"} {
				if host == vendor || strings.HasSuffix(host, "."+vendor) {
					return found{}, false
				}
			}
			if placeholder(host) {
				return found{}, false
			}
			return found{value: host, who: host, ties: []string{"domain:" + registrable(host)}, level: names}, true
		},
		lookup: "The page loads its counter from this server, which is yours: anybody who reads the page source sees where it runs.",
		advice: "Count the project's visits on a server that is the project's, or with a hosted service under an account of its own.",
	},
	{
		key: "analytics-site", name: "analytics site ID", level: ties,
		hints:  []string{"data-website-id", "data-site"},
		re:     regexp.MustCompile(`data-website-id=["']([0-9a-f-]{36})["']|usefathom[\s\S]{0,200}?data-site=["']([A-Z]{8})["']`),
		read:   one("analytics-site", ties, false),
		lookup: lookupMatch,
		advice: adviceSeparate,
	},
	{
		key: "ssh-key", name: "SSH public key", level: names,
		hints: []string{"ssh-ed25519 ", "ssh-rsa ", "ssh-dss ", "ecdsa-sha2-", "sk-ssh-", "sk-ecdsa-"},
		re:    regexp.MustCompile(`\b(ssh-ed25519|ssh-rsa|ssh-dss|ecdsa-sha2-nistp(?:256|384|521)|sk-ssh-ed25519@openssh\.com|sk-ecdsa-sha2-nistp256@openssh\.com) (AAAA[0-9A-Za-z+/]{40,}={0,3})(?:[ \t]+([^\s"'<>\\]{1,80}))?`),
		read: func(m []string) (found, bool) {
			fp, ok := sshFingerprint(m[2])
			if !ok {
				return found{}, false
			}
			f := found{value: fp, ties: []string{"ssh:" + fp}, level: names}
			if m[3] != "" && strings.Contains(m[3], "@") && !placeholder(m[3]) {
				f.who = m[3]
			}
			return f, true
		},
		lookup: "GitHub publishes every account's public keys at github.com/ and the account name followed by .keys, so a key can be matched to its account. The comment after a key is often the user and the machine it was made on.",
		advice: "Give the project a key of its own, made for it and used nowhere else.",
	},
	{
		key: "apple-team", name: "Apple developer team", level: ties,
		hints:  []string{"DEVELOPMENT_TEAM", "teamID", "appleTeamId", "TeamIdentifier"},
		re:     regexp.MustCompile(`DEVELOPMENT_TEAM\s*=\s*"?([A-Z0-9]{10})"?\s*;|<key>teamID</key>\s*<string>([A-Z0-9]{10})</string>|"appleTeamId"\s*:\s*"([A-Z0-9]{10})"`),
		read:   one("apple-team", ties, false),
		lookup: "The same ten letters are in every app the team signs, and codesign reads them out of any app on any Mac.",
		advice: "A team is an account with Apple, and an individual's team is in their legal name. A project that must stand apart from you needs a team that is not yours.",
	},
	{
		key: "apple-identity", name: "Apple signing identity", level: names,
		hints: []string{"Apple Development: ", "Apple Distribution: ", "iPhone Developer: ", "iPhone Distribution: ", "Developer ID Application: ", "Developer ID Installer: ", "Mac Developer: ", "3rd Party Mac Developer"},
		// Xcode writes the certificate's whole name into the project when
		// signing is set by hand, and an individual's certificate is in
		// their legal name.
		re: regexp.MustCompile(`(?:Apple Development|Apple Distribution|iPhone Developer|iPhone Distribution|Developer ID Application|Developer ID Installer|Mac Developer|3rd Party Mac Developer (?:Application|Installer)): ([^()"\n\r]{2,64}?) \(([A-Z0-9]{10})\)`),
		read: func(m []string) (found, bool) {
			if placeholder(m[1]) {
				return found{}, false
			}
			return found{value: m[0], who: m[1], ties: []string{"apple-team:" + strings.ToLower(m[2]), "person:" + strings.ToLower(m[1])}, level: names}, true
		},
		lookup: "It is the name on the certificate, an individual's own, and the team after it is in every app the certificate signs.",
		advice: "Leave signing to Xcode's automatic setting, which keeps the certificate's name out of the project, and sign a project that must stand apart with a team that is not yours.",
	},
	{
		key: "expo", name: "Expo account", level: names,
		hints:  []string{"\"owner\""},
		re:     regexp.MustCompile(`"owner"\s*:\s*"([a-z0-9][a-z0-9_-]{0,38})"`),
		read:   account("expo", nil),
		only:   named("app.json", "app.config.json"),
		lookup: "The owner is the Expo account the app is built and published under, and its page lists the rest.",
		advice: adviceSeparate,
	},
	{
		key: "deploy-config", name: "deployment account", level: ties,
		hints:  []string{"orgId", "siteId", "u.expo.dev", "projectId"},
		re:     regexp.MustCompile(`"orgId"\s*:\s*"((?:team_)?[A-Za-z0-9]{16,32})"|"siteId"\s*:\s*"([0-9a-f-]{36})"|u\.expo\.dev/([0-9a-f-]{36})|"projectId"\s*:\s*"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`),
		read:   one("deploy-config", ties, false),
		lookup: "It is the account or the project with the hosting company, the same in everything deployed from it. " + lookupMatch,
		advice: "These files are for your own machine. Keep .vercel and .netlify out of what you publish, and the account out of the configuration.",
	},
	{
		key: "bitcoin", name: "Bitcoin address", level: ties,
		hints: []string{"bc1", "bitcoin"},
		re:    regexp.MustCompile(`\b(bc1[ac-hj-np-z02-9]{11,71})\b|bitcoin:([13][a-km-zA-HJ-NP-Z1-9]{25,34})\b`),
		read: func(m []string) (found, bool) {
			if m[1] != "" && bech32OK(m[1]) {
				return found{value: m[1], ties: []string{"bitcoin:" + strings.ToLower(m[1])}, level: ties}, true
			}
			if m[2] != "" && bitcoinLegacy(m[2]) {
				return found{value: m[2], ties: []string{"bitcoin:" + m[2]}, level: ties}, true
			}
			return found{}, false
		},
		lookup: "Every payment to it and from it is public, and so is every other address the same wallet spends with.",
		advice: "Give the project an address, and a wallet, of its own.",
	},
	{
		key: "ethereum", name: "Ethereum address", level: ties,
		hints:  []string{"ethereum:", "ETH", "eth"},
		re:     regexp.MustCompile(`ethereum:(0x[0-9a-fA-F]{40})\b|\b(?i:eth(?:ereum)?(?:\s+address)?)\s*[:=]?\s*(0x[0-9a-fA-F]{40})\b`),
		read:   one("ethereum", ties, false),
		lookup: "Every transaction it has made is public, and names like ENS often point at it.",
		advice: "Give the project an address, and a wallet, of its own.",
	},
}

// registrable is the part of a host that somebody registered: the last
// two labels, or three under a country's second level.
func registrable(host string) string {
	labels := strings.Split(strings.TrimSuffix(host, "."), ".")
	if len(labels) <= 2 {
		return host
	}
	n := 2
	second := labels[len(labels)-2]
	if len(labels[len(labels)-1]) == 2 && (second == "co" || second == "com" || second == "org" || second == "net" || second == "ac" || second == "gov" || second == "edu") {
		n = 3
	}
	return strings.Join(labels[len(labels)-n:], ".")
}
