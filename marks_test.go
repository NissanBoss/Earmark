package main

import (
	"strings"
	"testing"
)

// A Mapbox token is a line of JSON in base64, with the user name in it.
func TestAMapboxTokenNamesItsUser(t *testing.T) {
	dir := project(t, map[string]string{
		"dist/assets/index-3f9a.js": `const map=new Map({accessToken:"` + mapboxToken("ana-riverside") + `"});`,
	})
	c := check(t, dir)
	c.only(t, "mapbox")
	if got := c.find("mapbox")[0].found.who; got != "ana-riverside" {
		t.Errorf("named %q", got)
	}
	// The token in Mapbox's own examples is nobody's project.
	check(t, project(t, map[string]string{"a.js": `t="` + mapboxToken("examples") + `"`})).only(t)
}

func TestAClerkKeyNamesTheDomainInProduction(t *testing.T) {
	live := "pk_" + "live_" + b64("clerk.ana-riverside.dev$")
	test := "pk_" + "test_" + b64("loved-mole-12.clerk.accounts.dev$")
	c := check(t, project(t, map[string]string{"main.js": `a="` + live + `";b="` + test + `"`}))
	got := c.find("clerk")
	if len(got) != 2 {
		t.Fatalf("found %d Clerk keys", len(got))
	}
	for _, sg := range got {
		switch {
		case strings.HasPrefix(sg.found.value, "pk_live"):
			if sg.found.who != "clerk.ana-riverside.dev" || sg.found.level != names {
				t.Errorf("the live key came out as %+v", sg.found)
			}
		default:
			if sg.found.level != ties {
				t.Errorf("a development key names a slug Clerk chose, not anybody: %+v", sg.found)
			}
		}
	}
}

// A Stripe key and a Clerk key share a prefix; only one of them is
// base64 of a host.
func TestAStripeKeyIsNotTakenForClerk(t *testing.T) {
	key := "pk_" + "live_" + "51Hx" + strings.Repeat("Ab3", 30)
	c := check(t, project(t, map[string]string{"pay.js": `Stripe("` + key + `")`}))
	c.only(t, "stripe")
}

func TestPlaceholdersAreLeftAlone(t *testing.T) {
	c := check(t, project(t, map[string]string{
		"index.html": `<script>gtag('config', 'G-XXXXXXXXXX'); ga('create','UA-XXXXX-Y');</script>
<ins data-ad-client="ca-pub-0000000000000000"></ins>
<a href="mailto:you@example.com">mail</a> <a href="https://github.com/your-username">me</a>`,
	}))
	c.only(t)
}

// The number in the middle of a Universal Analytics ID is the account,
// and every property of it shares it.
func TestTwoPropertiesOfOneAnalyticsAccountAreTied(t *testing.T) {
	side := project(t, map[string]string{"index.html": `ga('create', 'UA-4815162-3', 'auto');`})
	main := project(t, map[string]string{"index.html": `ga('create', 'UA-4815162-1', 'auto');`})
	out := check(t, side, main).text(true)
	contains(t, out, "[SHARED  ]")
	contains(t, out, "Google Analytics account 4*****2")
}

// AdSense on a site and AdMob in an app are the same publisher number.
func TestAdSenseAndAdMobAreOnePublisher(t *testing.T) {
	site := project(t, map[string]string{"index.html": `<script src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-4815162342108153"></script>`})
	app := project(t, map[string]string{"app/src/main/AndroidManifest.xml": `<meta-data android:name="com.google.android.gms.ads.APPLICATION_ID" android:value="ca-app-pub-4815162342108153~2718281828"/>`})
	c := check(t, site, app)
	if len(c.find("adsense")) != 1 {
		t.Fatal("the AdSense number was not found")
	}
	contains(t, c.text(true), "Google publisher")
}

func TestAVercelDeploymentNamesItsScope(t *testing.T) {
	c := check(t, project(t, map[string]string{"README.md": "Preview: https://shop-a1b2c3d4e-ana-riverside-projects.vercel.app and https://shop.vercel.app"}))
	var scoped, plain int
	for _, sg := range c.find("vercel") {
		if sg.found.who == "ana-riverside-projects" {
			scoped++
		} else if sg.found.who == "" {
			plain++
		}
	}
	if scoped != 1 || plain != 1 {
		t.Errorf("scoped %d, plain %d", scoped, plain)
	}
}

func TestAWorkersAddressNamesTheAccount(t *testing.T) {
	c := check(t, project(t, map[string]string{"api.js": `fetch("https://img-proxy.anariver.workers.dev/x")`}))
	c.only(t, "workers")
	if got := c.find("workers")[0].found.who; got != "anariver" {
		t.Errorf("named %q", got)
	}
}

// A github.com link inside a minified bundle is nearly always a library's
// own; a file loaded from somebody's repository is not.
func TestGitHubInABundleIsTheLibrarysUnlessItIsLoaded(t *testing.T) {
	c := check(t, project(t, map[string]string{
		"dist/vendor.js": `throw Error("see https://github.com/somelib/somelib/issues/12")`,
		"dist/app.js":    `img.src="https://raw.githubusercontent.com/anariver/assets/main/logo.png"`,
	}))
	c.only(t, "github")
	if got := c.find("github")[0].found.who; got != "anariver" {
		t.Errorf("named %q", got)
	}
}

// Two sites on one theme both link to its author, and are not one
// person's for it.
func TestAThemesAuthorDoesNotTieTwoSites(t *testing.T) {
	footer := `<footer>Theme by <a href="https://github.com/mmistakes/minimal-mistakes">mmistakes</a></footer>`
	a := project(t, map[string]string{"index.html": footer})
	b := project(t, map[string]string{"index.html": footer})
	out := check(t, a, b).text(true)
	if strings.Contains(out, "SHARED") {
		t.Errorf("a theme's credit tied two sites:\n%s", out)
	}
}

func TestAProfileLinkAndAThemeSetting(t *testing.T) {
	c := check(t, project(t, map[string]string{
		"_config.yml":      "title: Notes\ntwitter_username: anariver\ngithub_username: anariver\n",
		"about/index.html": `<a href="https://www.linkedin.com/in/ana-riverside-1a2b3c/">LinkedIn</a>`,
	}))
	c.only(t, "config-handle", "profile:linkedin")
}

// A token copied from a real session says who was signed in. A Supabase
// key says which project, and the service key gets past every rule.
func TestWhatATokenSays(t *testing.T) {
	session := jwt(map[string]any{"sub": "8f2a", "email": "ana.riverside@proton.me"})
	anon := jwt(map[string]any{"iss": "supabase", "ref": "abcdefghijklmnopqrst", "role": "anon"})
	service := jwt(map[string]any{"iss": "supabase", "ref": "abcdefghijklmnopqrst", "role": "service_role"})
	c := check(t, project(t, map[string]string{
		"test/fixture.json": `{"token":"` + session + `"}`,
		"src/db.ts":         `createClient(url, "` + anon + `"); const admin = "` + service + `";`,
	}))
	var person, proj, secret int
	for _, sg := range c.find("jwt") {
		switch {
		case sg.found.who == "ana.riverside@proton.me":
			person++
		case sg.found.note != "":
			secret++
		default:
			proj++
		}
	}
	if person != 1 || proj != 1 || secret != 1 {
		t.Errorf("person %d, project %d, service key %d", person, proj, secret)
	}
}

func TestBitcoinAddressesAreCheckedNotGuessed(t *testing.T) {
	// The test vectors of BIP 173 and of the Bitcoin wiki.
	good := "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"
	legacy := "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2"
	c := check(t, project(t, map[string]string{"donate.html": "Send to " + good + " or bitcoin:" + legacy}))
	if n := len(c.find("bitcoin")); n != 2 {
		t.Errorf("found %d of the two real addresses", n)
	}
	bad := strings.Replace(good, "w508", "w509", 1)
	badLegacy := strings.Replace(legacy, "BvB", "BvC", 1)
	check(t, project(t, map[string]string{"donate.html": "Send to " + bad + " or bitcoin:" + badLegacy})).only(t)
}

// Libraries fetched by a package manager are other people's code.
func TestNodeModulesAreNotRead(t *testing.T) {
	dir := project(t, map[string]string{
		"node_modules/somelib/dist/index.js": `const t="` + mapboxToken("somelib-author") + `"`,
		"src/index.js":                       `console.log(1)`,
	})
	c := check(t, dir)
	c.only(t)
	contains(t, c.text(false), "node_modules")
}

// What it found is masked unless --reveal is given, and keeps its shape.
func TestTheReportIsMasked(t *testing.T) {
	dir := project(t, map[string]string{"a.js": `t="` + mapboxToken("ana-riverside") + `"`})
	out := check(t, dir).text(false)
	if strings.Contains(out, "ana-riverside") {
		t.Errorf("the user name is in the report:\n%s", out)
	}
	contains(t, out, "a*a-r*******e")
	c := check(t, dir)
	c.show.reveal = true
	contains(t, c.text(false), "ana-riverside")
}

func TestAnIDKeepsItsShape(t *testing.T) {
	cases := map[string]string{
		"UA-4815162-3":         "UA-4*****2-3",
		"pub-4815162342108153": "pub-4**************3",
		"G-7QK2LM9XZP":         "G-7********P",
	}
	for in, want := range cases {
		if got := (shower{}).id(in); got != want {
			t.Errorf("%s came out as %s", in, got)
		}
	}
	if got := (shower{}).id(mapboxToken("ana")); got != "pk.***" {
		t.Errorf("a long token came out as %s", got)
	}
}

// A host is that host only when it starts the address: GitHub's own
// documentation is not an account called en, and Dropbox is not x.com.
func TestAHostInsideAnotherIsNotIt(t *testing.T) {
	c := check(t, project(t, map[string]string{
		"README.md": "See https://docs.github.com/en/pages and https://www.dropbox.com/s/abc123/file.zip and https://developers.facebook.com/docs/sharing",
	}))
	c.only(t)
}

// What a certificate says is masked, and the labels this puts in front of
// it are not.
func TestACertificateIsMaskedAndStillReads(t *testing.T) {
	if got := (shower{}).text("name Ana Riverside, city Valencia, country ES"); got != "name A*a R*******e, city V******a, country E*" {
		t.Errorf("came out as %q", got)
	}
}
