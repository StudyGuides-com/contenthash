package contenthash

import (
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// nfkc is NFKC with x/text's decomposition and canonical composition done
// here. x/text v0.32.0 looks composition pairs up by the low 16 bits of each
// character, so a character above U+FFFF followed by a combining mark can
// compose as if it were a BMP letter: U+E0067 TAG LATIN SMALL LETTER G with
// U+0301 becomes U+01F5. Its decomposition and combining classes are correct
// for every code point.
func nfkc(s string) string {
	return compose(norm.NFKD.String(s))
}

// compose is canonical composition (UAX #15) of decomposed, canonically
// ordered text.
func compose(s string) string {
	out := make([]rune, 0, len(s))
	starter := -1 // index in out of the last starter
	last := -1    // combining class of the last character after it; -1 for none
	for _, r := range s {
		ccc := combiningClass(r)
		if starter >= 0 && (last == -1 || (last != 0 && last < ccc)) {
			if c, ok := combine(out[starter], r); ok {
				out[starter] = c
				continue
			}
		}
		if ccc == 0 {
			starter, last = len(out), -1
		} else {
			last = ccc
		}
		out = append(out, r)
	}
	return string(out)
}

func combiningClass(r rune) int {
	var b [utf8.UTFMax]byte
	n := utf8.EncodeRune(b[:], r)
	return int(norm.NFD.Properties(b[:n]).CCC())
}

// Hangul syllable composition, from the Unicode standard's algorithm.
const (
	hangulS = 0xAC00
	hangulL = 0x1100
	hangulV = 0x1161
	hangulT = 0x11A7
	countL  = 19
	countV  = 21
	countT  = 28
	countN  = countV * countT
)

func combine(a, b rune) (rune, bool) {
	switch {
	case a >= hangulL && a < hangulL+countL && b >= hangulV && b < hangulV+countV:
		return hangulS + ((a-hangulL)*countV+(b-hangulV))*countT, true
	case a >= hangulS && a < hangulS+countL*countN && (a-hangulS)%countT == 0 && b > hangulT && b < hangulT+countT:
		return a + (b - hangulT), true
	}
	c, ok := compositions[[2]rune{a, b}]
	return c, ok
}
