package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// certificate makes a self-signed certificate the way keytool does, with
// whatever was typed in for the name.
func certificate(t *testing.T, subject pkix.Name) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: subject, NotBefore: time.Unix(1e9, 0), NotAfter: time.Unix(4e9, 0)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// pkcs7 wraps a certificate the way META-INF/CERT.RSA has it: signed data
// with the certificate in it, and nothing that this reads besides.
func pkcs7(t *testing.T, cert []byte) []byte {
	t.Helper()
	var signed struct {
		Version      int
		Digests      asn1.RawValue
		Content      asn1.RawValue
		Certificates asn1.RawValue `asn1:"optional,tag:0"`
	}
	signed.Version = 1
	signed.Digests = asn1.RawValue{Class: 0, Tag: 17, IsCompound: true}
	inner, _ := asn1.Marshal(struct{ Type asn1.ObjectIdentifier }{asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}})
	signed.Content = asn1.RawValue{FullBytes: inner}
	signed.Certificates = asn1.RawValue{Class: 2, Tag: 0, IsCompound: true, Bytes: cert}
	body, err := asn1.Marshal(signed)
	if err != nil {
		t.Fatal(err)
	}
	// Marshal writes FullBytes as they are and ignores the tag, so the
	// [0] around the signed data is put on by hand.
	out, err := asn1.Marshal(struct {
		Type    asn1.ObjectIdentifier
		Content asn1.RawValue
	}{asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}, asn1.RawValue{Class: 2, Tag: 0, IsCompound: true, Bytes: body}})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func lp(parts ...[]byte) []byte {
	var body []byte
	for _, p := range parts {
		body = append(body, p...)
	}
	out := make([]byte, 4, 4+len(body))
	binary.LittleEndian.PutUint32(out, uint32(len(body)))
	return append(out, body...)
}

// withSigningBlock puts an APK signature scheme v2 block, holding one
// signer with this certificate, in front of the zip's central directory.
func withSigningBlock(t *testing.T, apk, cert []byte) []byte {
	t.Helper()
	cd := centralDirectory(apk)
	eocd := bytes.LastIndex(apk, []byte("PK\x05\x06"))
	signed := lp(lp(), lp(lp(cert)), lp())
	signer := lp(signed, lp(), lp([]byte("key")))
	value := lp(signer)
	pair := make([]byte, 12)
	binary.LittleEndian.PutUint64(pair, uint64(4+len(value)))
	binary.LittleEndian.PutUint32(pair[8:], 0x7109871a)
	pair = append(pair, value...)
	size := uint64(len(pair) + 8 + 16)
	block := make([]byte, 8)
	binary.LittleEndian.PutUint64(block, size)
	block = append(block, pair...)
	tail := make([]byte, 8)
	binary.LittleEndian.PutUint64(tail, size)
	block = append(append(block, tail...), "APK Sig Block 42"...)
	out := append(append(append([]byte{}, apk[:cd]...), block...), apk[cd:]...)
	binary.LittleEndian.PutUint32(out[eocd+len(block)+16:], uint32(cd+len(block)))
	return out
}

func writeFile(t *testing.T, name string, body []byte) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAnAPKSaysWhoSignedIt(t *testing.T) {
	cert := certificate(t, pkix.Name{CommonName: "Ana Riverside", Locality: []string{"Valencia"}, Country: []string{"ES"}})
	apk := zipOf(t, map[string][]byte{
		"AndroidManifest.xml": []byte("manifest"),
		"META-INF/CERT.RSA":   pkcs7(t, cert),
	})
	for name, body := range map[string][]byte{"jar scheme": apk, "v2 block": withSigningBlock(t, zipOf(t, map[string][]byte{"AndroidManifest.xml": []byte("m")}), cert)} {
		t.Run(name, func(t *testing.T) {
			c := check(t, writeFile(t, "app-release.apk", body))
			got := c.find("signing")
			if len(got) != 1 {
				t.Fatalf("found %d signing certificates", len(got))
			}
			f := got[0].found
			if f.level != names || !strings.Contains(f.who, "name Ana Riverside") || !strings.Contains(f.who, "city Valencia") {
				t.Errorf("came out as %+v", f)
			}
			sum := sha256.Sum256(cert)
			if f.value != "SHA-256 "+strings.ToUpper(hex.EncodeToString(sum[:])) {
				t.Errorf("fingerprint %s", f.value)
			}
		})
	}
}

// The debug key is made once per machine, so every debug build from it is
// signed the same: a mark of the machine.
func TestADebugBuildIsSignedByItsMachine(t *testing.T) {
	cert := certificate(t, pkix.Name{CommonName: "Android Debug", Organization: []string{"Android"}, Country: []string{"US"}})
	c := check(t, writeFile(t, "app-debug.apk", zipOf(t, map[string][]byte{"META-INF/CERT.RSA": pkcs7(t, cert)})))
	f := c.find("signing")[0].found
	if f.level != listed || !strings.Contains(f.note, "debug key") {
		t.Errorf("came out as %+v", f)
	}
}

func profileFile(team, id string, devices ...string) []byte {
	var d strings.Builder
	for _, u := range devices {
		d.WriteString("<string>" + u + "</string>")
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict>` +
		`<key>AppIDName</key><string>Notes</string><key>Name</key><string>Notes Ad Hoc</string>` +
		`<key>ProvisionedDevices</key><array>` + d.String() + `</array>` +
		`<key>TeamIdentifier</key><array><string>` + id + `</string></array>` +
		`<key>TeamName</key><string>` + team + `</string></dict></plist>`
	return append([]byte{0x30, 0x80, 0x06, 0x09}, plist...)
}

func TestAProvisioningProfileNamesTheTeamAndTheDevices(t *testing.T) {
	ipa := zipOf(t, map[string][]byte{"Payload/Notes.app/embedded.mobileprovision": profileFile("Ana Riverside", "AB12CD34EF", "00008110-001A2B3C4D5E6F70", "00008030-000F1E2D3C4B5A69")})
	c := check(t, writeFile(t, "Notes.ipa", ipa))
	got := c.find("profile")
	if len(got) != 1 {
		t.Fatalf("found %d profiles", len(got))
	}
	f := got[0].found
	if f.who != "team Ana Riverside" || !strings.Contains(f.note, "2 devices") {
		t.Errorf("came out as %+v", f)
	}
}

func TestAnNPMPackageIsReadInside(t *testing.T) {
	var buf bytes.Buffer
	z := gzip.NewWriter(&buf)
	tw := tar.NewWriter(z)
	body := []byte(`module.exports="` + mapboxToken("ana-riverside") + `"`)
	tw.WriteHeader(&tar.Header{Name: "package/dist/index.js", Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg})
	tw.Write(body)
	tw.Close()
	z.Close()
	c := check(t, writeFile(t, "notes-1.0.0.tgz", buf.Bytes()))
	got := c.find("mapbox")
	if len(got) != 1 || !strings.HasSuffix(got[0].places[0], "notes-1.0.0.tgz > package/dist/index.js, line 1") {
		t.Fatalf("found %+v", got)
	}
}

// A token baked into an app is a string in its binary, ASCII or UTF-16.
func TestTheStringsOfABinary(t *testing.T) {
	key := "AIza" + "SyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q"
	var wide []byte
	for _, c := range mapboxToken("ana-riverside") {
		wide = append(wide, byte(c), 0)
	}
	blob := append(append(append([]byte{0x7f, 'E', 'L', 'F', 0, 0, 1, 2}, key...), 0, 0, 0, 9), wide...)
	c := check(t, writeFile(t, "libnative.so", append(blob, 0, 0)))
	c.only(t, "google-key", "mapbox")
	if p := c.find("mapbox")[0].places[0]; !strings.HasSuffix(p, "libnative.so, in its strings") {
		t.Errorf("placed at %s", p)
	}
}

func TestAnSSHKeySaysWhoseItIs(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	blob := append(lpBE([]byte("ssh-ed25519")), lpBE(pub)...)
	line := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob) + " ana@thinkpad"
	c := check(t, project(t, map[string]string{"deploy/authorized_keys": line + "\n"}))
	f := c.find("ssh-key")[0].found
	sum := sha256.Sum256(blob)
	if f.who != "ana@thinkpad" || f.value != "SHA256:"+base64.RawStdEncoding.EncodeToString(sum[:]) {
		t.Errorf("came out as %+v", f)
	}
}

func lpBE(b []byte) []byte {
	out := make([]byte, 4)
	binary.BigEndian.PutUint32(out, uint32(len(b)))
	return append(out, b...)
}

// An OpenPGP key carries its names and addresses as plain text.
func TestAnOpenPGPKeyNamesItsOwner(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	// A version 4 EdDSA key: version, time, algorithm 22, the curve's OID,
	// and the point.
	key := []byte{4, 0x60, 0, 0, 0, 22, 9, 0x2b, 0x06, 0x01, 0x04, 0x01, 0xda, 0x47, 0x0f, 0x01, 0x01, 0x07}
	key = append(key, append([]byte{0x40}, pub...)...)
	uid := []byte("Ana Riverside <ana.riverside@proton.me>")
	packets := append(append([]byte{0xc6, byte(len(key))}, key...), append([]byte{0xcd, byte(len(uid))}, uid...)...)
	armored := "-----BEGIN PGP PUBLIC KEY BLOCK-----\nComment: made for a test\n\n" + base64.StdEncoding.EncodeToString(packets) + "\n=abcd\n-----END PGP PUBLIC KEY BLOCK-----\n"
	c := check(t, project(t, map[string]string{"KEYS.asc": armored}))
	got := c.find("pgp")
	if len(got) != 1 {
		t.Fatalf("found %d keys", len(got))
	}
	h := sha1.New()
	h.Write([]byte{0x99, 0, byte(len(key))})
	h.Write(key)
	f := got[0].found
	if f.who != string(uid) || f.value != strings.ToUpper(hex.EncodeToString(h.Sum(nil))) {
		t.Errorf("came out as %+v", f)
	}
}

// What it could not read is said, and a report with a gap in it never
// calls itself clear.
func TestABrokenArchiveIsAGap(t *testing.T) {
	c := check(t, writeFile(t, "build.zip", []byte("PK\x03\x04 not really a zip")))
	r := c.report(false)
	if len(r.Gaps) == 0 {
		t.Fatal("a broken zip made no gap")
	}
	for _, f := range r.Findings {
		if f.Tag == "CLEAR   " {
			t.Error("a report with something unread called itself clear")
		}
	}
}
