package main

// OpenPGP public keys. A key carries its user IDs in the clear, name and
// address, and its fingerprint is what keyservers and GitHub file it
// under. A project that publishes its signing key publishes whose it is.

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"
)

var armored = regexp.MustCompile(`(?s)-----BEGIN PGP PUBLIC KEY BLOCK-----(.*?)-----END PGP PUBLIC KEY BLOCK-----`)

var pgpMark = &mark{
	key: "pgp", name: "OpenPGP public key", level: names,
	lookup: "The names and addresses on a key are in it as plain text, and keyservers and GitHub look keys up by their fingerprint.",
	advice: "Sign a project that must stand apart with a key made for it, carrying the project's name and an address of its own.",
}

func scanPGP(s *side, where, text string) {
	if !strings.Contains(text, "BEGIN PGP PUBLIC KEY BLOCK") {
		return
	}
	starts := lineStarts(text)
	for _, idx := range armored.FindAllStringSubmatchIndex(text, -1) {
		body := dearmor(text[idx[2]:idx[3]])
		if body == nil {
			continue
		}
		for _, key := range pgpKeys(body) {
			s.add(pgpMark, key, where+", "+position(starts, idx[0]))
		}
	}
}

// dearmor drops the header lines, which have a colon in them and base64
// never does, and the checksum line that starts with "=", and decodes the
// rest.
func dearmor(block string) []byte {
	var b strings.Builder
	for _, l := range strings.Split(block, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "=") {
			break
		}
		if l != "" && !strings.Contains(l, ":") {
			b.WriteString(l)
		}
	}
	raw, err := base64.StdEncoding.DecodeString(b.String())
	if err != nil {
		return nil
	}
	return raw
}

// pgpKeys walks the packets: a public key, then its user IDs, then the
// next key. Only version 4 fingerprints are worked out, which is what
// nearly every key in use is.
func pgpKeys(b []byte) []found {
	var out []found
	var cur *found
	for len(b) > 0 {
		tag, body, rest, ok := packet(b)
		if !ok {
			break
		}
		b = rest
		switch tag {
		case 6:
			if cur != nil {
				out = append(out, *cur)
			}
			cur = &found{value: "key", level: names}
			if len(body) > 0 && body[0] == 4 {
				h := sha1.New()
				h.Write([]byte{0x99, byte(len(body) >> 8), byte(len(body))})
				h.Write(body)
				fp := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
				cur.value = fp
				cur.ties = append(cur.ties, "pgp:"+fp)
			}
		case 13:
			if cur != nil {
				uid := strings.TrimSpace(string(body))
				if uid != "" {
					if cur.who == "" {
						cur.who = uid
					} else {
						cur.who += "; " + uid
					}
					if i, j := strings.Index(uid, "<"), strings.Index(uid, ">"); i >= 0 && j > i {
						cur.ties = append(cur.ties, "email:"+strings.ToLower(uid[i+1:j]))
					}
				}
			}
		}
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

// packet reads one OpenPGP packet header, old or new format.
func packet(b []byte) (tag int, body, rest []byte, ok bool) {
	if len(b) < 2 || b[0]&0x80 == 0 {
		return 0, nil, nil, false
	}
	var n, head int
	if b[0]&0x40 != 0 {
		tag = int(b[0] & 0x3f)
		switch l := int(b[1]); {
		case l < 192:
			n, head = l, 2
		case l < 224:
			if len(b) < 3 {
				return 0, nil, nil, false
			}
			n, head = (l-192)<<8+int(b[2])+192, 3
		case l == 255:
			if len(b) < 6 {
				return 0, nil, nil, false
			}
			n, head = int(b[2])<<24|int(b[3])<<16|int(b[4])<<8|int(b[5]), 6
		default:
			// Partial lengths are for data packets, never for keys.
			return 0, nil, nil, false
		}
	} else {
		tag = int(b[0]>>2) & 0x0f
		switch b[0] & 3 {
		case 0:
			n, head = int(b[1]), 2
		case 1:
			if len(b) < 3 {
				return 0, nil, nil, false
			}
			n, head = int(b[1])<<8|int(b[2]), 3
		case 2:
			if len(b) < 5 {
				return 0, nil, nil, false
			}
			n, head = int(b[1])<<24|int(b[2])<<16|int(b[3])<<8|int(b[4]), 5
		default:
			return 0, nil, nil, false
		}
	}
	if n < 0 || head+n > len(b) {
		return 0, nil, nil, false
	}
	return tag, b[head : head+n], b[head+n:], true
}
