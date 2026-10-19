package main

// Two projects at once: the one that must stand apart, and one that is
// known to be yours. What they share is the tie itself, whatever it is,
// and the lookup services are not needed for it: anybody with both in
// front of them sees the same account in each.

import (
	"fmt"
	"sort"
	"strings"
)

// What each family of ties is called when it is shown on its own.
var tieNames = map[string]string{
	"mapbox": "Mapbox account", "clerk": "Clerk instance", "stripe": "Stripe key", "stripe-link": "Stripe payment link",
	"ga": "Google Analytics account", "ga4": "Google Analytics property", "gtm": "Google Tag Manager container",
	"google-ads": "Google Ads account", "google-pub": "Google publisher", "google-key": "Google API key",
	"google-project": "Google Cloud project", "firebase": "Firebase project", "sentry": "Sentry organisation",
	"sentry-project": "Sentry project", "supabase": "Supabase project", "person": "name", "auth0": "Auth0 tenant",
	"algolia": "Algolia application", "cloudinary": "Cloudinary account", "mailchimp": "Mailchimp account",
	"mailchimp-name": "Mailchimp account", "amazon": "Amazon Associates tag", "gravatar": "Gravatar address",
	"disqus": "Disqus site", "facebook-pixel": "Meta Pixel", "facebook-app": "Facebook app",
	"facebook-admins": "Facebook admins", "twitter": "X account", "hotjar": "Hotjar site", "clarity": "Clarity project",
	"tiktok": "TikTok pixel", "linkedin": "LinkedIn account", "cloudflare-analytics": "Cloudflare analytics site",
	"cloudflare-account": "Cloudflare account", "workers": "Cloudflare Workers account", "github": "GitHub account",
	"gitlab": "GitLab account", "vercel-scope": "Vercel account", "vercel": "Vercel project", "netlify": "Netlify site",
	"replit": "Replit account", "host": "app", "bucket": "storage bucket", "registry": "container registry account",
	"donation": "payment page", "email": "email address", "phone": "phone number", "telegram": "Telegram account",
	"discord": "Discord invite", "service": "support or booking account", "chat-widget": "chat widget",
	"domain": "domain", "analytics-site": "analytics site", "ssh": "SSH key", "pgp": "OpenPGP key",
	"apple-team": "Apple developer team", "expo": "Expo account", "deploy-config": "deployment account",
	"bitcoin": "Bitcoin address", "ethereum": "Ethereum address", "cert": "signing key", "device": "device",
}

// shared finds the ties that are in both projects.
func (c *run) shared() (Finding, bool) {
	there := map[string][]*sighting{}
	for _, sg := range c.sides[1].sightings() {
		for _, t := range sg.found.ties {
			there[t] = append(there[t], sg)
		}
	}
	here := map[string][]*sighting{}
	var both []string
	for _, sg := range c.sides[0].sightings() {
		for _, t := range sg.found.ties {
			if _, ok := there[t]; !ok {
				continue
			}
			if _, ok := here[t]; !ok {
				both = append(both, t)
			}
			here[t] = append(here[t], sg)
		}
	}
	if len(both) == 0 {
		return Finding{}, false
	}
	sort.Strings(both)
	var lines []string
	for n, t := range both {
		if n > 0 {
			lines = append(lines, "")
		}
		family, value, _ := strings.Cut(t, ":")
		label := tieNames[family]
		if label == "" {
			label = family
		}
		shown := c.show.id(value)
		if family == "person" || family == "email" || strings.HasSuffix(label, "account") || family == "domain" {
			shown = c.show.text(value)
		}
		lines = append(lines, label+" "+shown)
		lines = append(lines, "  here:  "+firstPlace(here[t]))
		lines = append(lines, "  there: "+firstPlace(there[t]))
	}
	title := "One thing is in both, and ties them together"
	if len(both) > 1 {
		title = fmt.Sprintf("%d things are in both, and tie them together", len(both))
	}
	return Finding{Tag: "SHARED  ", Weight: Shared, Title: title, Lines: lines,
		Advice: "Anybody who has both in front of them sees the same account in each, and needs no lookup service to do it. Each of these wants replacing in the project that must stand apart, with an account that is its own."}, true
}

func firstPlace(list []*sighting) string {
	n := 0
	for _, sg := range list {
		n += len(sg.places)
	}
	p := list[0].places[0]
	if n > 1 {
		p += fmt.Sprintf(", and %d more", n-1)
	}
	return p
}
