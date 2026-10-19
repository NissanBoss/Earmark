package main

// Identifiers that carry something inside them. A Mapbox token is a bit of
// JSON with your user name in it, a Clerk key is your sign-in domain, a
// Supabase key is a signed note naming your project. None of it is
// encrypted, only encoded, and the reader below is all it takes.

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
)

// unbase64 reads base64 in either alphabet, padded or not. Tokens are
// written all four ways in the wild.
func unbase64(s string) ([]byte, bool) {
	s = strings.TrimRight(s, "=")
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.RawStdEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, true
		}
	}
	return nil, false
}

// mapboxUser is the "u" of a Mapbox token: the user name of the account
// that made it.
func mapboxUser(body string) (string, bool) {
	raw, ok := unbase64(body)
	if !ok {
		return "", false
	}
	var t struct {
		U string `json:"u"`
	}
	if json.Unmarshal(raw, &t) != nil || t.U == "" {
		return "", false
	}
	return t.U, true
}

// clerkHost is the sign-in host a Clerk publishable key spells out: the
// base64 of the host and a dollar sign.
func clerkHost(body string) (string, bool) {
	raw, ok := unbase64(body)
	if !ok || len(raw) < 5 || raw[len(raw)-1] != '$' {
		return "", false
	}
	host := string(raw[:len(raw)-1])
	if !strings.Contains(host, ".") || strings.ContainsAny(host, " /\\\"'<>") {
		return "", false
	}
	return strings.ToLower(host), true
}

// jwtClaims reads the middle part of a JWT without checking its signature,
// which is all that is needed to see what it says.
func jwtClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	raw, ok := unbase64(parts[1])
	if !ok {
		return nil
	}
	var claims map[string]any
	if json.Unmarshal(raw, &claims) != nil {
		return nil
	}
	return claims
}

// sshFingerprint is what ssh-keygen -l prints for a public key, which is
// how anyone holding the key would name it.
func sshFingerprint(blob string) (string, bool) {
	raw, err := base64.StdEncoding.DecodeString(blob)
	if err != nil || len(raw) < 20 {
		return "", false
	}
	sum := sha256.Sum256(raw)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), true
}

const base58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// bitcoinLegacy checks the base58 checksum of a 1... or 3... address. A
// string of the right letters is not an address; one in four billion of
// them has the right checksum by chance.
func bitcoinLegacy(s string) bool {
	n := new(big.Int)
	for _, c := range s {
		i := strings.IndexRune(base58, c)
		if i < 0 {
			return false
		}
		n.Mul(n, big.NewInt(58))
		n.Add(n, big.NewInt(int64(i)))
	}
	raw := n.Bytes()
	for _, c := range s {
		if c != '1' {
			break
		}
		raw = append([]byte{0}, raw...)
	}
	if len(raw) != 25 {
		return false
	}
	first := sha256.Sum256(raw[:21])
	second := sha256.Sum256(first[:])
	return string(second[:4]) == string(raw[21:])
}

// bech32OK checks the checksum of a bc1 address, by BIP 173 and 350.
func bech32OK(s string) bool {
	const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	s = strings.ToLower(s)
	sep := strings.LastIndexByte(s, '1')
	if sep < 1 || sep+7 > len(s) {
		return false
	}
	var values []int
	for _, c := range s[:sep] {
		values = append(values, int(c)>>5)
	}
	values = append(values, 0)
	for _, c := range s[:sep] {
		values = append(values, int(c)&31)
	}
	for _, c := range s[sep+1:] {
		i := strings.IndexRune(charset, c)
		if i < 0 {
			return false
		}
		values = append(values, i)
	}
	gen := []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := 1
	for _, v := range values {
		top := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ v
		for i := 0; i < 5; i++ {
			if top>>i&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk == 1 || chk == 0x2bc830a3
}

// vercelScope is the account or team at the end of a deployment address,
// project-hash-scope.vercel.app, after the nine letters of the hash. The
// branch form, project-git-branch-scope, is left alone: a branch name has
// dashes in it too, and where it stops and the scope starts cannot be told.
// A production address, project.vercel.app, has no scope at all.
func vercelScope(label string) (string, bool) {
	parts := strings.Split(label, "-")
	for i := 1; i+1 < len(parts); i++ {
		p := parts[i]
		if len(p) == 9 && isLowerAlnum(p) && hasDigit(p) {
			return strings.Join(parts[i+1:], "-"), true
		}
	}
	return "", false
}

func isLowerAlnum(s string) bool {
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return s != ""
}

func hasDigit(s string) bool {
	return strings.ContainsAny(s, "0123456789")
}
