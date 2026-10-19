# Changelog

## v1.1

- **The labels in front of what a certificate says are not masked.** v1
  masked them with the rest, and a signing certificate came out as
  `n**e A*a R*******e, c*****y E*`; it now reads `name A*a R*******e,
  country E*`, the name masked and the label not.

## v1

The first release.

Says what ties a project to you: which identifiers in a built site, an
app, a package or a repository belong to an account, which of them name
the account outright, and which two of your projects share.

- **Identifiers that name the account:** the user name inside a Mapbox
  token, the domain inside a Clerk key, the team at the end of a Vercel
  deployment address, the account subdomain of a Cloudflare Worker, the
  person a session token was made for, the name and city on an Android
  app's signing certificate, the team and the devices in an iOS
  provisioning profile, the names and addresses on an OpenPGP key, the
  comment on an SSH key, and the accounts in the addresses a project loads
  from, links to or takes payments through.
- **Identifiers the lookup services index:** Google Analytics, Tag
  Manager, AdSense and AdMob, which share one publisher number, the Meta
  Pixel and others, which SpyOnWeb, SameID and BuiltWith list with every
  other site that carries them.
- **Identifiers that only match:** a Sentry organisation, a Firebase or
  Supabase project, a Google Cloud project number, a Stripe key, and the
  like, which tie together whatever carries them.
- **Two projects at once:** with `--against`, what a project shares with
  another one of yours, which is the tie itself.
- **Inside containers:** zips, APKs, IPAs, browser extensions, npm and
  Python packages, and the strings of the binaries in them, without
  unpacking anything.
- **Checked against real files** on every push, made by Vite, npm,
  keytool, jarsigner, ssh-keygen and gpg, with the keys' fingerprints
  compared with what their own tools print; and against personal sites,
  the most visited sites and published apps, counted on GitHub's machines.
- **It only reads:** nothing is written, nothing is looked up or sent, no
  other program is run, with tests that read the source to keep it that
  way. What it finds is masked unless `--reveal` is given.
