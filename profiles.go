package main

// Profiles on other sites, and the configuration that fills them in. On a
// site that is meant to be yours they are the point; on one that must
// stand apart they are the plainest tie there is, and the commonest: in
// forty personal sites tried first, a LinkedIn profile was on ten. Themes
// keep them in their configuration, twitter_username and the like, and
// a theme cloned for a second site brings the first one's along.

import (
	"regexp"
	"strings"
)

type platform struct {
	name     string
	hint     string
	re       string // the handle is the first group that matched
	reserved []string
}

var platforms = []platform{
	{"LinkedIn", "linkedin.com/", `linkedin\.com/(?:in|pub)/([A-Za-z0-9_%-]{3,100})`, nil},
	{"LinkedIn company", "linkedin.com/company/", `linkedin\.com/company/([A-Za-z0-9_%-]{2,100})`, nil},
	{"X", ".com/", `(?:twitter|x)\.com/(?:#!/)?([A-Za-z0-9_]{1,15})\b`, []string{"intent", "share", "home", "search", "hashtag", "i", "settings", "explore", "login", "signup", "privacy", "tos", "about", "jobs", "notifications", "messages", "compose", "widgets", "twitterapi", "x", "twitter", "en", "web"}},
	{"Instagram", "instagram.com/", `instagram\.com/([A-Za-z0-9_.]{1,30})\b`, []string{"p", "explore", "accounts", "reel", "reels", "stories", "about", "developer", "legal", "direct", "tv", "instagram"}},
	{"Facebook", "facebook.com/", `facebook\.com/([A-Za-z0-9.]{5,50})\b`, []string{"sharer", "share", "dialog", "plugins", "tr", "groups", "pages", "events", "login", "help", "privacy", "policies", "watch", "marketplace", "gaming", "profile", "photo", "story", "permalink", "facebook", "business", "legal"}},
	{"YouTube", "youtube.com/", `youtube\.com/(?:@|c/|user/|channel/)([A-Za-z0-9_.-]{2,60})`, nil},
	{"TikTok", "tiktok.com/@", `tiktok\.com/@([A-Za-z0-9_.]{2,24})`, nil},
	{"Medium", "medium.com", `medium\.com/@([A-Za-z0-9_.-]{1,40})|\b([a-z0-9-]{2,40})\.medium\.com\b`, []string{"www", "help", "policy", "blog", "cdn-images-1", "miro", "cdn"}},
	{"Bluesky", "bsky.app/profile/", `bsky\.app/profile/([A-Za-z0-9.:-]{3,80})`, nil},
	{"Threads", "threads.", `threads\.(?:net|com)/@([A-Za-z0-9_.]{1,30})`, nil},
	{"Reddit", "reddit.com/u", `reddit\.com/u(?:ser)?/([A-Za-z0-9_-]{3,20})`, nil},
	{"Stack Overflow", "stackoverflow.com/users/", `stackoverflow\.com/users/(\d{2,10})`, nil},
	{"DEV", "dev.to/", `\bdev\.to/([A-Za-z0-9_]{2,40})\b`, []string{"t", "search", "top", "enter", "about", "faq", "settings", "new", "privacy", "terms", "api"}},
	{"Kaggle", "kaggle.com/", `kaggle\.com/([A-Za-z0-9_-]{2,40})\b`, []string{"competitions", "datasets", "code", "models", "discussions", "learn", "docs", "c", "kernels", "account", "static", "general", "terms", "privacy"}},
	{"Hugging Face", "huggingface.co/", `huggingface\.co/([A-Za-z0-9_.-]{2,40})\b`, []string{"datasets", "spaces", "models", "docs", "blog", "papers", "join", "login", "settings", "pricing", "api", "tasks", "learn", "posts", "chat", "collections", "organizations", "new"}},
	{"LeetCode", "leetcode.", `leetcode\.(?:com|cn)/(?:u/)?([A-Za-z0-9_-]{2,40})/?["'\s]`, []string{"problems", "contest", "discuss", "explore", "studyplan", "problemset", "accounts", "tag", "company", "u"}},
	{"CodePen", "codepen.io/", `codepen\.io/([A-Za-z0-9_-]{2,40})\b`, []string{"pen", "trending", "search", "challenges", "spark", "features", "login", "signup", "about", "support"}},
	{"Dribbble", "dribbble.com/", `dribbble\.com/([A-Za-z0-9_-]{2,40})\b`, []string{"shots", "jobs", "designers", "tags", "search", "learn", "pro", "signup", "session", "following"}},
	{"Behance", "behance.net/", `behance\.net/([A-Za-z0-9_-]{2,40})\b`, []string{"gallery", "search", "joblist", "galleries", "assets"}},
	{"Keybase", "keybase.io/", `keybase\.io/([A-Za-z0-9_]{2,16})\b`, []string{"docs", "blog", "download", "team", "_"}},
	{"npm", "npmjs.com/~", `npmjs\.com/~([A-Za-z0-9_.-]{2,214})`, nil},
	{"PyPI", "pypi.org/user/", `pypi\.org/user/([A-Za-z0-9_.-]{2,50})`, nil},
	{"Twitch", "twitch.tv/", `twitch\.tv/([A-Za-z0-9_]{4,25})\b`, []string{"directory", "videos", "settings", "subscriptions", "downloads", "jobs", "p", "search"}},
	{"ORCID", "orcid.org/", `orcid\.org/(\d{4}-\d{4}-\d{4}-\d{3}[\dX])`, nil},
	{"Google Scholar", "scholar.google.", `scholar\.google\.[a-z.]{2,6}/citations\?(?:[^"'\s]*&(?:amp;)?)?user=([A-Za-z0-9_-]{12})`, nil},
	{"ResearchGate", "researchgate.net/profile/", `researchgate\.net/profile/([A-Za-z0-9_%-]{3,80})`, nil},
	{"Codeberg", "codeberg.org/", `codeberg\.org/([A-Za-z0-9_.-]{2,40})\b`, []string{"explore", "user", "repo", "admin", "assets", "codeberg"}},
	{"Telegram", "telegram.me/", `telegram\.me/([A-Za-z][A-Za-z0-9_]{4,31})\b`, []string{"joinchat", "share"}},
}

// Configuration keys that themes fill in with the owner's handles.
var configHandle = regexp.MustCompile(`(?m)^\s*"?(github|twitter|x|linkedin|instagram|facebook|youtube|mastodon|keybase|gitlab|medium|stackoverflow|bluesky|scholar|orcid|telegram|reddit|kaggle|dribbble|behance|twitch|tiktok|threads|codepen|devto|hackernews|npm|pypi)(?:_username|_user|_handle|_id)?"?\s*[:=]\s*["']?([A-Za-z0-9_.@:-]{2,60})["']?\s*,?\s*$`)

func init() {
	for _, p := range platforms {
		p := p
		reserved := map[string]bool{}
		for _, r := range p.reserved {
			reserved[r] = true
		}
		family := "profile:" + strings.ToLower(strings.ReplaceAll(p.name, " ", "-"))
		marks = append(marks, &mark{
			key: family, name: p.name + " profile", level: names,
			hints:  []string{p.hint},
			re:     regexp.MustCompile(hostStart + "(?:" + p.re + ")"),
			read:   account(family, reserved),
			lookup: "It is a link to a profile, which says whose it is to anybody who follows it.",
			advice: adviceProfile,
		})
	}
	marks = append(marks, &mark{
		key: "config-handle", name: "handle in the configuration", level: names,
		re: configHandle,
		read: func(m []string) (found, bool) {
			v := strings.TrimPrefix(m[2], "@")
			l := strings.ToLower(v)
			if placeholder(v) || docsAccounts[l] || l == "true" || l == "false" || l == "null" || l == "none" || strings.Contains(v, "${") {
				return found{}, false
			}
			return found{value: m[1] + ": " + v, who: v, ties: []string{"profile:" + strings.ToLower(m[1]) + ":" + l}, level: names}, true
		},
		only:   configOnly,
		lookup: "Themes put these into every page they make, as links and as tags for sharing.",
		advice: adviceProfile,
	})
	marks = append(marks,
		&mark{
			key: "disqus-config", name: "Disqus site", level: names,
			hints:  []string{"disqus"},
			re:     regexp.MustCompile(`disqus_shortname\s*[:=]\s*["']([a-z0-9][a-z0-9-]{1,60})["']|(?m)^\s*disqus(?:_shortname)?\s*:\s*\n?\s*(?:shortname\s*:\s*)?["']?([a-z0-9][a-z0-9-]{1,60})["']?\s*$`),
			read:   account("disqus", map[string]bool{"true": true, "false": true}),
			lookup: lookupAddress,
			advice: adviceSeparate,
		},
		&mark{
			key: "formspree", name: "Formspree form", level: ties,
			hints: []string{"formspree.io/"},
			re:    regexp.MustCompile(`formspree\.io/(?:f/)?([A-Za-z0-9]{6,12})\b|formspree\.io/([A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[a-z]{2,})`),
			read: func(m []string) (found, bool) {
				if m[2] != "" {
					return found{value: m[2], who: m[2], ties: []string{"email:" + strings.ToLower(m[2])}, level: names}, true
				}
				if placeholder(m[1]) {
					return found{}, false
				}
				return found{value: m[1], ties: []string{"formspree:" + m[1]}, level: ties}, true
			},
			lookup: "A form sends what is typed into it to the address of the account that made it.",
			advice: adviceSeparate,
		},
	)
}

const adviceProfile = "If it is yours, a project that must stand apart cannot link to it. Themes and templates bring their author's handles along, so the configuration is worth reading line by line."

func configOnly(name string) bool {
	switch strings.ToLower(filepathExt(name)) {
	case ".yml", ".yaml", ".toml", ".json", ".ini", ".cfg", ".env", ".properties":
		return true
	}
	return false
}

func filepathExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 && !strings.ContainsAny(name[i:], `/\`) {
		return name[i:]
	}
	return ""
}
