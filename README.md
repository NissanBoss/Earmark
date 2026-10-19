# Earmark

What ties a project to you: which identifiers in a built site, an app, a
package or a repository belong to an account, which of them name the
account outright, and which two of your projects share.

An earmark is the notch a farmer cuts in an animal's ear to say whose it
is. It is small, it is meant to be read, and it goes wherever the animal
goes.

```
    [SHARED  ] One thing is in both, and ties them together
              Google publisher 4**************3
                here:  ~/projects/side-project/dist/index.html, line 1
                there: ~/projects/my-blog/index.html, line 1

    [NAMES   ] It carries a Mapbox token that names its account
              pk.***, which names a*a-r*******e
                ~/projects/side-project/dist/assets/index-3f9a.js, line 1

              The user name is written inside the token, as base64 of a line
              of JSON, for anybody who opens the page source.

    [NAMES   ] It carries a Vercel deployment by name
              s**p-a*******e-a*a-r********s-p******s.vercel.app
                ~/projects/side-project/dist/assets/index-3f9a.js, line 1

    [NAMES   ] It carries a Cloudflare Workers address by name
              img.a******r.workers.dev
                ~/projects/side-project/dist/assets/index-3f9a.js, line 1

    [LISTED  ] It carries a Google AdSense publisher ID, which the lookup services
              index
              pub-4**************3
```

That is a project made up for this page, cut short; every finding is
followed by who can use it and what to do about it, left out here.
`earmark side-project/ --against my-blog/` printed it.

## The problem

You start a project that must not be tied to you: a site under another
name, an app you would rather your employer did not find, something you
publish anonymously because of what it says. You make a new GitHub
account, a new address, a new name. And then you build it with the tools
you always use, signed in to the accounts you always have.

None of what those accounts leave in the build is a secret, so nothing
looks at it. Every one of them is a fact about an account, published on
purpose:

- **A Mapbox token has your user name in it.** It is base64 of a line of
  JSON, `{"u":"your-name",...}`, in every map you publish. Of 260 public
  Mapbox tokens read from public code on GitHub, the name inside was
  exactly the repository owner's in 60, and somebody else's user name in
  the rest.
- **An AdSense number is you.** Google allows one AdSense account to a
  person, it has to be on the page for the ads to work, and SpyOnWeb,
  SameID and BuiltWith will give anybody every other site that carries it,
  including sites that dropped it years ago. Bellingcat has followed these
  numbers to whole networks of sites. AdMob, in apps, shows the same
  number.
- **An Android app carries whatever was typed into keytool.** The
  certificate it is signed with has the name the key was made with, and
  the same key signs every app the developer makes for as long as they
  make it. Of 200 apps their developers publish on GitHub, 178 carry a
  name, a person's or an organisation's, and 89 a city as well.
- **A deployment address ends with its account.** A Vercel preview is
  project-hash-*scope*.vercel.app, where the scope is the account or team
  it was deployed from. A Cloudflare Worker is
  name.*your-subdomain*.workers.dev, one subdomain to an account. A Clerk
  key is your sign-in domain in base64.
- **A test build of an iOS app names its team and lists its phones.** The
  provisioning profile inside an ad hoc build has the team, which for one
  person is their legal name, and the hardware IDs of every device it may
  run on.
- **A copied site brings its author along.** In the calibration below, the
  same AdSense number turned up on seventeen personal sites of different
  people, most likely copies of one site: a copy goes on carrying its
  author's ads, and its author's name with them.

Each of these is known to somebody. Together they are a map back to you,
and nothing draws it before you publish.

## What already exists

- [SpyOnWeb](https://spyonweb.com), [SameID](https://sameid.net),
  [DNSlytics](https://dnslytics.com) and
  [BuiltWith](https://builtwith.com) go the other way: give them an
  identifier and they give you every site that carries it.
  [Bellingcat's guide](https://www.bellingcat.com/resources/how-tos/2015/07/23/unveiling-hidden-connections-with-google-analytics-ids/)
  explains how investigators use them. They are the reason an identifier
  matters, and they answer about somebody else's site, after it is up.
- [gitleaks](https://github.com/gitleaks/gitleaks),
  [trufflehog](https://github.com/trufflesecurity/trufflehog) and GitHub's
  own secret scanning look for secrets, and skip a publishable key on
  purpose, because anybody may see it. Whether it is yours is not their
  question.
- [Kerbside](https://github.com/NissanBoss/Kerbside), from this same
  series, reads a build for the secrets and source maps it should not
  serve. Earmark reads the same build for something else: not whether a
  key can be abused, but whose it is.
- [Accent](https://github.com/NissanBoss/Accent),
  [Tell](https://github.com/NissanBoss/Tell) and
  [Alibi](https://github.com/NissanBoss/Alibi), also from this series,
  tie a project to you by how you write code and prose and by when you
  commit. Earmark ties it by the accounts it runs on.

Nothing reads your own project for the accounts in it and tells you, before
it is published, which of them are you.

## What it reads

```bash
earmark dist/
earmark side-project/ --against my-blog/
```

A folder is walked for text, the files a site, an app's sources or a
repository are made of, and for containers. A zip, an APK or an AAB, an
IPA, a browser extension, an npm or Python package is read inside without
unpacking anything, and so are the strings of the binaries in them, where
an app keeps the tokens compiled into it. What it finds comes in three
kinds.

**Names.** Identifiers that say outright what the account is called, or
who is behind it: the user name inside a Mapbox token, the domain inside a
Clerk key, the team at the end of a Vercel deployment, the subdomain of a
Cloudflare Worker, the tenant of Auth0, the account of a Mailchimp form, an
Amazon Associates tag, a Disqus site, the person a session token was made
for, the name and city on an app's signing certificate, the team and the
devices of an iOS provisioning profile, an Apple signing identity, the
names and addresses on an OpenPGP key, the user and machine on an SSH key,
a Gravatar hash, and the accounts in the addresses a project loads from,
links to or takes money through: GitHub, payment pages, profiles and the
handles a theme keeps in its configuration.

**Listed.** Identifiers the lookup services index, so that one of them is
enough to find the rest: Google Analytics accounts and measurement IDs, Tag
Manager containers, Google Ads, AdSense and AdMob, which share one
publisher number, the Meta Pixel, TikTok's and LinkedIn's tags, and the
signing certificate of an Android app, which APKMirror shows for every app
it hosts.

**Ties.** Identifiers of an account that say nothing alone and everything
to whoever has another project of yours: a Sentry organisation, a Firebase
or Supabase project, a Google Cloud project number, which is in every
sign-in client and notification sender the project has, a Google API key,
a Stripe key, a Cloudflare account, the accounts of the chat widget and the
analytics, a storage bucket, an app's own address, a Bitcoin or Ethereum
address.

With `--against`, it reads a second project too, one that is known to be
yours, and says what the two share. That needs no lookup service: anybody
with both in front of them sees the same account in each.

Placeholders are left alone (`UA-XXXXX-Y`, `ca-pub-0000000000000000`), and
so are the accounts documentation uses, Mapbox's examples among them.
Libraries fetched by a package manager are not read, `node_modules` and
`Pods`, because every identifier in them is their authors'. A link to a
theme's author or a library's, each of which the calibration found on four or
more personal sites, is left out by name, and inside an app, a binary or a
licence file no link to an account is counted at all: in two hundred apps,
those were the libraries'.

The exit code is 1 when something names an account, is listed, or is
shared with the other project, 0 when nothing did, 2 when the files could
not be opened, and 3 when nothing given could be read.

## The calibration

Every rule here was tried on real files before it was kept, and most of
them were changed by it. The comments in the code say which files changed
which rule.

**Projects made with the tools people use**, on every push
([`tools.yml`](.github/workflows/tools.yml)): a site built by Vite with
its keys in the environment the way its documentation says to put them, a
package made by `npm pack`, an app signed by `jarsigner` with a key
`keytool` made, another with a key made the way Android Studio makes its
debug key, an SSH key from `ssh-keygen` and an OpenPGP key from `gpg`. The
keys' fingerprints are compared with what `keytool`, `ssh-keygen` and
`gpg` themselves print for them, not with this program's own idea of the
answer.

**Thousands of real projects**, by hand and once a month
([`calibrate.yml`](.github/workflows/calibrate.yml)), downloaded and
counted on GitHub's machines. The last run, 19 October 2026:

- **1,214 personal sites on GitHub Pages**, the user site of each owner.
  964 carry something that names an account. Of the GitHub accounts in
  them, 1,341 were the site's owner and 1,139 somebody else's, the
  projects and people a site links to. After GitHub, the commonest were an
  email address on 218, a Google Analytics measurement ID on 149, LinkedIn
  on 148, X on 129, Discord invites on 98 and AdSense on 48.
- **Two sites of one person.** 134 of the owners have a second site, a
  project page. 55 of the pairs are tied, and 15 by something besides the
  GitHub account they are both served from: a profile, an address, an
  analytics site, a Workers subdomain.
- **Two sites of different people.** Of 1,214 pairs of neighbours in the
  list, 15 are tied. Nine of them by one AdSense number, on seventeen
  sites of different people, most likely copies of one site; the rest by a
  third account both link to, an invite to the same Discord server, the
  same address.
- **1,204 of the 2,000 most visited sites**, by the
  [Tranco](https://tranco-list.eu) list; the rest refused a plain request.
  Tag Manager is on 431, a Google Analytics measurement ID on 180, the old
  Analytics on 84, a Google API key on 53, Sentry on 51, AdSense on 48, the
  Meta Pixel on 32, a Mapbox token on 4. And on more than one of these
  domains at once: 26 Tag Manager containers, 7 AdSense publishers, 4
  Analytics accounts and 151 social profiles, the same companies under
  other names.
- **200 Android apps** their developers publish in GitHub releases. Every
  certificate was read: 178 name a person or an organisation, 89 with a
  city as well; 14 are signed with the debug key of the machine that built
  them; 9 carry nothing true. Besides the signing, a Google API key is in
  30, a Google Cloud project number in 20, a Firebase project in 18, a
  Workers subdomain in 13 and a Vercel deployment in 13.
- **300 Mapbox tokens in public code** on GitHub, found through its code
  search: 260 read as tokens, and the user name inside was exactly the
  repository owner's in 60 of them.

What the calibration taught, all of it in the code:

- A github.com link in a minified bundle is a library's: bundles carry them
  in their error messages and their licences. In scripts, only the
  addresses that load something count, raw.githubusercontent.com and
  jsDelivr's; an address on github.io in a script is a library's
  documentation, normalize.css's and slick's in every copy.
- A host is that host only when it starts the address. docs.github.com/en
  is not an account called en, and dropbox.com/s/ is not x.com.
- Themes and icon fonts load from their authors' accounts: Academicons,
  Chirpy, Just the Docs, Feather, the Nerfies template, each on four or
  more of the personal sites. They are left out by name.
- Inside an app, the accounts it links to are its libraries': JetBrains in
  31 of the 200 apps, Kotlin in 30, and AboutLibraries, which makes the
  credits screen, put its author in 29.
- Sites print the visitor's user agent into the page. The first run's
  agent had this repository's address in it, and came back from twenty of
  the most visited sites as a GitHub account.
- A key made with nothing true in its name is how it should be done.
  Termux signs with "APK Signer" from "Earth", and it is reported as a key
  and nothing more.
- A vendor's Google Cloud project number is on fourteen of the most
  visited sites, which are not one company's, and is left out.
- facebook.com/sharer.php is not a profile, and neither is
  instagram.com/embed.js.

## What it cannot do

- **It does not look anything up.** There is no network code in it, so it
  cannot say whether an identifier is already in SpyOnWeb's index, which
  Cloud project a Google key belongs to, or whose a Gravatar hash is. It
  says which kinds the lookup services index, and leaves the asking to you.
- **It only sees what is in the files.** The ties outside them are not
  read: the domain's registration, the pair of nameservers Cloudflare
  usually gives every domain of one account, the certificates that certificate
  transparency logs publish with every name on them, the address the
  server answers from, the headers it sends.
- **It does not know which accounts are yours.** A link to a friend's
  GitHub is reported like a link to your own; on personal sites, nearly
  half the GitHub accounts found were somebody else's. With `--against`, only what
  is in both counts, which is the question that matters.
- **It does not read how you write, code or work.** Style, vocabulary and
  the hours you commit tie projects together too; Tell, Accent and Alibi
  are about those.
- **Pictures and media are not read**, and what they carry, a camera's
  serial, a location, an author, is Baggage's question. Neither are
  compressed or encrypted binaries: an app from the App Store is encrypted,
  and its strings cannot be read without decrypting it.
- **The list is long, not complete.** Every service with an account and an
  identifier in the page is a possible tie, and new ones keep appearing. A
  clean report covers the ones listed in the source, and says so.

## It only reads

- **Nothing is written.** Not a report file, not a cache. There are tests
  that read the source and fail the build if a call that writes, deletes or
  renames anything appears in it.
- **Nothing is looked up or sent.** No key is checked with its provider, no
  identifier with a lookup service: there is no network code in it, and a
  test fails the build if any appears. It does not run other programs
  either.
- **What it finds is masked.** Names and identifiers are cut to their
  first and last letters, so the report can be pasted somewhere; the home
  folder is shown as `~`. `--reveal` shows both.
- **A gap is not a pass.** A file it could not read, an archive it could
  not open, an entry that is encrypted: each is said, first, and a report
  with a gap in it never calls itself clear.
- **Doing it right is not a finding.** A signing key made with nothing true
  in its name, as Termux makes its, is reported as a key and nothing else.
  A placeholder is not an account. A theme's author is not yours.
- **Exact, not statistical.** Every tie is an identifier that is the same
  in both places, character for character. Nothing is a likelihood.

## Running it

Download the package for your system from the Releases page, unpack it,
and run the binary. There is nothing to install.

```bash
earmark dist/
```

```bash
earmark --reveal side-project/ --against my-blog/
```

Windows marks anything that came from the internet, so the first run may
bring up a blue "Windows protected your PC" box. Right click the zip,
Properties, tick Unblock at the bottom, and then extract it. Building it
yourself avoids the question.

## Building

Needs [Go](https://go.dev) and nothing else. No dependencies at all.

```bash
go build -o earmark .
```

```bash
go test ./...
```

```bash
sh testdata/tools.sh /tmp/tools
EARMARK_TOOLS=/tmp/tools go test -run TestMadeByTools -v
```

The tests build the projects they check, apps, keys and profiles
included, so what makes each case is in the test and not in a binary
nobody can read. Tokens in them are put together from pieces, so that no
line of the repository has the shape of a real one.

## Licence

MIT. Do what you like with it.
