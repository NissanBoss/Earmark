package main

// Who signed it. An Android app carries the certificate it was signed
// with, and the certificate carries whatever name was typed into keytool
// when the key was made, often a real one, with a city and a country. The
// same key signs every app of the developer, so its fingerprint ties them
// together, and APKMirror shows it for any app on Google Play. An iOS
// build carries its provisioning profile: the team, which for one person
// is their legal name, and the phones it was allowed to run on.

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// apkSigningBlock finds the certificates in the signing block that APK
// signature schemes 2 and 3 put just before the central directory.
func apkSigningBlock(data []byte, cdOffset int) [][]byte {
	if cdOffset < 32 || cdOffset > len(data) || string(data[cdOffset-16:cdOffset]) != "APK Sig Block 42" {
		return nil
	}
	size := int(binary.LittleEndian.Uint64(data[cdOffset-24:]))
	start := cdOffset - size - 8
	if size < 24 || start < 0 {
		return nil
	}
	pairs := data[start+8 : cdOffset-24]
	var certs [][]byte
	for len(pairs) >= 12 {
		n := int(binary.LittleEndian.Uint64(pairs))
		if n < 4 || n > len(pairs)-8 {
			break
		}
		id := binary.LittleEndian.Uint32(pairs[8:])
		value := pairs[12 : 8+n]
		pairs = pairs[8+n:]
		switch id {
		case 0x7109871a, 0xf05368c0, 0x1b93ad61:
			certs = append(certs, signerCerts(value)...)
		}
	}
	return certs
}

// signerCerts walks the length-prefixed lists of a v2 or v3 block: signers,
// each with its signed data, which has digests and then certificates.
func signerCerts(value []byte) [][]byte {
	var certs [][]byte
	signers, ok := prefixed(value)
	if !ok {
		return nil
	}
	for len(signers) > 0 {
		signer, rest, ok := next(signers)
		if !ok {
			break
		}
		signers = rest
		signed, _, ok := next(signer)
		if !ok {
			continue
		}
		_, after, ok := next(signed) // the digests
		if !ok {
			continue
		}
		list, _, ok := next(after)
		for ok && len(list) > 0 {
			var cert []byte
			cert, list, ok = next(list)
			if ok {
				certs = append(certs, cert)
			}
		}
	}
	return certs
}

func prefixed(b []byte) ([]byte, bool) {
	v, _, ok := next(b)
	return v, ok
}

// next splits off one item with its four-byte length in front.
func next(b []byte) (item, rest []byte, ok bool) {
	if len(b) < 4 {
		return nil, nil, false
	}
	n := int(binary.LittleEndian.Uint32(b))
	if n > len(b)-4 {
		return nil, nil, false
	}
	return b[4 : 4+n], b[4+n:], true
}

// pkcs7Certs reads the certificates out of a signature file in META-INF,
// the way the first scheme, the jar one, keeps them.
func pkcs7Certs(der []byte) [][]byte {
	var info struct {
		Type    asn1.ObjectIdentifier
		Content asn1.RawValue `asn1:"explicit,tag:0"`
	}
	if _, err := asn1.Unmarshal(der, &info); err != nil {
		return nil
	}
	var signed struct {
		Version      int
		Digests      asn1.RawValue
		Content      asn1.RawValue
		Certificates asn1.RawValue `asn1:"optional,tag:0"`
	}
	if _, err := asn1.Unmarshal(info.Content.Bytes, &signed); err != nil {
		return nil
	}
	var certs [][]byte
	rest := signed.Certificates.Bytes
	for len(rest) > 0 {
		var one asn1.RawValue
		var err error
		rest, err = asn1.Unmarshal(rest, &one)
		if err != nil {
			break
		}
		certs = append(certs, one.FullBytes)
	}
	return certs
}

// certFound turns a signing certificate into what it says. The debug key
// Android Studio makes is the same for every debug build from one machine,
// which makes it a mark of the machine rather than of a person.
func certFound(der []byte) (found, bool) {
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return found{}, false
	}
	sum := sha256.Sum256(der)
	fp := strings.ToUpper(hex.EncodeToString(sum[:]))
	f := found{value: "SHA-256 " + fp, ties: []string{"cert:" + fp}, level: listed}
	subject := cert.Subject
	if len(subject.CommonName) > 0 && subject.CommonName == "Android Debug" {
		f.note = "It is the debug key Android Studio made on the machine that built this: every debug build from that machine is signed with the same one."
		return f, true
	}
	var parts []string
	told := false
	add := func(label string, values ...string) {
		for _, v := range values {
			if v = strings.TrimSpace(v); v != "" && !emptyName[strings.ToLower(v)] {
				parts = append(parts, label+" "+v)
				told = told || label != "country"
			}
		}
	}
	add("name", subject.CommonName)
	add("organisation", subject.Organization...)
	add("unit", subject.OrganizationalUnit...)
	add("city", subject.Locality...)
	add("region", subject.Province...)
	add("country", subject.Country...)
	// A country on its own says where, not who; a key made with nothing
	// true in it says neither, and is how it should be done.
	if told {
		f.who = strings.Join(parts, ", ")
		f.level = names
	} else if len(parts) > 0 {
		f.note = "The certificate says nothing about who made it but the country, " + strings.TrimPrefix(parts[0], "country ") + "."
	}
	for _, e := range cert.EmailAddresses {
		f.ties = append(f.ties, "email:"+strings.ToLower(e))
	}
	if cn := strings.TrimSpace(subject.CommonName); cn != "" && !emptyName[strings.ToLower(cn)] {
		f.ties = append(f.ties, "person:"+strings.ToLower(cn))
	}
	return f, true
}

// emptyName is what people type into keytool when they mean to say
// nothing, which is the right thing to type. Termux signs with "APK
// Signer" from "Earth".
var emptyName = map[string]bool{
	"unknown": true, "apk signer": true, "signer": true, "android": true, "developer": true, "release": true,
	"releasekey": true, "release key": true, "key": true, "key0": true, "upload": true, "upload key": true,
	"test": true, "testkey": true, "debug": true, "earth": true, "company": true, "organization": true,
	"organisation": true, "org": true, "none": true, "n/a": true, "na": true, "my company": true, "app": true,
	"application": true, "default": true, "sample": true, "example": true, "xx": true, "world": true,
	"local": true, "localhost": true, "internet": true, "web": true, "-": true, ".": true,
}

var signingMark = &mark{
	key: "signing", name: "signing certificate", level: listed,
	lookup: "The same key signs every app of the developer, and APKMirror shows its fingerprint for every app on Google Play. What the certificate says was typed into keytool when the key was made.",
	advice: "A key is made once and kept for the life of an app, so an app already out cannot change it. A new project that must stand apart from you needs a key made for it, with nothing true typed into the name.",
}

var profileMark = &mark{
	key: "profile", name: "provisioning profile", level: names,
	lookup: "The team is in every app it signs. For one person the team name is their legal name, and the devices listed are phones and tablets by their hardware IDs.",
	advice: "A build made for testing carries the profile it was made with. Share builds made for the App Store or TestFlight, which carry no device list, from a team that is the project's.",
}

var plistString = regexp.MustCompile(`<key>(TeamName|Name|AppIDName)</key>\s*<string>([^<]{1,120})</string>`)
var plistTeam = regexp.MustCompile(`<key>TeamIdentifier</key>\s*<array>\s*<string>([A-Z0-9]{10})</string>`)
var plistDevices = regexp.MustCompile(`(?s)<key>ProvisionedDevices</key>\s*<array>(.*?)</array>`)
var plistData = regexp.MustCompile(`(?s)<key>DeveloperCertificates</key>\s*<array>(.*?)</array>`)
var plistString1 = regexp.MustCompile(`<string>([^<]+)</string>`)
var plistData1 = regexp.MustCompile(`(?s)<data>(.*?)</data>`)

// profile reads an embedded.mobileprovision, which is a signed envelope
// around a property list in plain XML.
func (c *run) profile(s *side, where string, data []byte) bool {
	start := bytes.Index(data, []byte("<?xml"))
	end := bytes.Index(data, []byte("</plist>"))
	if start < 0 || end < start {
		return false
	}
	text := string(data[start : end+8])
	f := found{level: names}
	var who []string
	for _, m := range plistString.FindAllStringSubmatch(text, -1) {
		if m[1] == "TeamName" {
			who = append(who, "team "+m[2])
			f.ties = append(f.ties, "person:"+strings.ToLower(m[2]))
		}
	}
	if m := plistTeam.FindStringSubmatch(text); m != nil {
		f.value = "team " + m[1]
		f.ties = append(f.ties, "apple-team:"+strings.ToLower(m[1]))
	}
	if m := plistDevices.FindStringSubmatch(text); m != nil {
		devices := plistString1.FindAllStringSubmatch(m[1], -1)
		for _, d := range devices {
			f.ties = append(f.ties, "device:"+strings.ToLower(d[1]))
		}
		if len(devices) > 0 {
			f.note = fmt.Sprintf("It lists %s by their hardware IDs, which tie it to anything else made for the same devices.", plural(len(devices), "device", "devices"))
		}
	}
	if m := plistData.FindStringSubmatch(text); m != nil {
		for _, d := range plistData1.FindAllStringSubmatch(m[1], -1) {
			der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(d[1]), ""))
			if err != nil {
				continue
			}
			if cert, err := x509.ParseCertificate(der); err == nil && cert.Subject.CommonName != "" {
				who = append(who, "certificate "+cert.Subject.CommonName)
			}
		}
	}
	if f.value == "" && len(who) == 0 {
		return false
	}
	if f.value == "" {
		f.value = "profile"
	}
	f.who = strings.Join(dedupe(who), ", ")
	s.add(profileMark, f, where)
	return true
}

func dedupe(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range list {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
