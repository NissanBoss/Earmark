#!/bin/sh
# Makes projects the way people make them, with the tools they use: a site
# built by Vite with its keys in the environment, a package made by npm
# pack, an app signed by jarsigner with a key keytool made, an SSH key from
# ssh-keygen and an OpenPGP key from gpg. Each line of expect.tsv says
# what must be found where, for TestMadeByTools, and for the keys what the
# tools themselves say their fingerprints are. Run by tools.yml on Ubuntu.

set -e
out="${1:?give a folder}"
mkdir -p "$out"
cd "$out"
: > expect.tsv

# Tokens shaped like real ones and belonging to nobody: a Mapbox token for
# the user earmark-test, a Clerk key for clerk.earmark-test.dev.
MAPBOX=$(python3 -c 'import base64,json;b=lambda s:base64.urlsafe_b64encode(s).decode().rstrip("=");print("pk."+b(json.dumps({"u":"earmark-test","a":"ckx12ab3c0001"}).encode())+".Qx7Kp2LmN9vR4sT1uW3yZ5")')
CLERK=$(python3 -c 'import base64;print("pk_live_"+base64.b64encode(b"clerk.earmark-test.dev$").decode().rstrip("="))')

# A site, built by Vite, with the keys coming in from the environment the
# way its documentation says to.
npm create vite@6 site -- --template vanilla >/dev/null 2>&1
cd site
npm install --silent >/dev/null 2>&1
cat > src/main.js <<'JS'
const map = { token: import.meta.env.VITE_MAPBOX_TOKEN }
const auth = { key: import.meta.env.VITE_CLERK_PUBLISHABLE_KEY }
const dsn = "https://0123456789abcdef0123456789abcdef@o4504123.ingest.sentry.io/4504999"
document.querySelector('#app').textContent = JSON.stringify({ map, auth, dsn }).length
JS
sed -i 's#</head>#<script async src="https://www.googletagmanager.com/gtag/js?id=G-7QK2LM9XZP"></script><script>gtag("config", "G-7QK2LM9XZP")</script></head>#' index.html
VITE_MAPBOX_TOKEN="$MAPBOX" VITE_CLERK_PUBLISHABLE_KEY="$CLERK" npx vite build >/dev/null
cd ..
printf 'site/dist\tmapbox\tearmark-test\n' >> expect.tsv
printf 'site/dist\tclerk\tclerk.earmark-test.dev\n' >> expect.tsv
printf 'site/dist\tsentry\to4504123\n' >> expect.tsv
printf 'site/dist\tga4\tG-7QK2LM9XZP\n' >> expect.tsv

# A package, made by npm pack.
mkdir -p pkg && cd pkg
printf '{"name":"earmark-sample","version":"1.0.0","main":"index.js"}\n' > package.json
printf 'module.exports = "%s"\n' "$MAPBOX" > index.js
npm pack --silent >/dev/null
cd ..
printf 'pkg/earmark-sample-1.0.0.tgz\tmapbox\tearmark-test\n' >> expect.tsv

# An app signed the jar way, with a key made by keytool, and another with a
# key made the way Android Studio makes its debug key.
mkdir -p app && cd app
printf 'manifest' > AndroidManifest.xml
zip -q release.apk AndroidManifest.xml
cp release.apk debug.apk
keytool -genkeypair -keystore release.jks -alias release -dname "CN=Earmark Test, L=Testville, C=ES" -storepass earmark -keypass earmark -keyalg RSA -keysize 2048 -validity 10000 >/dev/null 2>&1
jarsigner -keystore release.jks -storepass earmark release.apk release >/dev/null
keytool -genkeypair -keystore debug.jks -alias androiddebugkey -dname "CN=Android Debug,O=Android,C=US" -storepass android -keypass android -keyalg RSA -keysize 2048 -validity 10000 >/dev/null 2>&1
jarsigner -keystore debug.jks -storepass android debug.apk androiddebugkey >/dev/null
FP=$(keytool -list -v -keystore release.jks -storepass earmark -alias release | sed -n 's/.*SHA256: //p' | tr -d ':')
cd ..
printf 'app/release.apk\tsigning\tname Earmark Test, city Testville, country ES\n' >> expect.tsv
printf 'app/release.apk\tsigning\tSHA-256 %s\n' "$FP" >> expect.tsv
printf 'app/debug.apk\tsigning\tdebug key\n' >> expect.tsv

# An SSH key, with the fingerprint ssh-keygen gives it.
mkdir -p keys
ssh-keygen -q -t ed25519 -C "earmark@testbox" -N "" -f keys/id_ed25519
cp keys/id_ed25519.pub keys/authorized_keys
rm keys/id_ed25519 keys/id_ed25519.pub
SSHFP=$(ssh-keygen -lf keys/authorized_keys | cut -d' ' -f2)
printf 'keys\tssh-key\t%s\n' "$SSHFP" >> expect.tsv
printf 'keys\tssh-key\tearmark@testbox\n' >> expect.tsv

# An OpenPGP key, with the fingerprint gpg gives it.
export GNUPGHOME="$out/gnupg"
mkdir -p "$GNUPGHOME" && chmod 700 "$GNUPGHOME"
gpg --batch --quiet --passphrase '' --quick-gen-key "Earmark Test <earmark.test@users.noreply.github.com>" ed25519 sign never
gpg --batch --quiet --armor --export earmark.test@users.noreply.github.com > keys/KEYS.asc
PGPFP=$(gpg --batch --with-colons --fingerprint earmark.test@users.noreply.github.com | awk -F: '/^fpr/ {print $10; exit}')
rm -rf "$GNUPGHOME"
printf 'keys\tpgp\t%s\n' "$PGPFP" >> expect.tsv
printf 'keys\tpgp\tEarmark Test <earmark.test@users.noreply.github.com>\n' >> expect.tsv

cat expect.tsv
