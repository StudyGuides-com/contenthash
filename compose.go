package contenthash

import (
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// nfkc is NFKC, built from x/text's decomposition of single characters, with
// canonical ordering and composition done here. x/text v0.32.0 gets two
// things wrong for this purpose:
//
//   - Its composition looks pairs up by the low 16 bits of each character, so
//     a character above U+FFFF before a combining mark can compose as if it
//     were a BMP letter: U+E0067 TAG LATIN SMALL LETTER G with U+0301 becomes
//     U+01F5.
//   - Normalizing a whole string inserts U+034F COMBINING GRAPHEME JOINER after
//     30 non-starters in a row (UAX #15's Stream-Safe Text Format), which plain
//     NFKC doesn't.
//
// Its decomposition of a single character and its combining classes are
// correct for every code point, and one character never decomposes to 30
// non-starters.
func nfkc(s string) string {
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		if r < utf8.RuneSelf {
			runes = append(runes, r) // ASCII doesn't decompose
			continue
		}
		runes = append(runes, []rune(norm.NFKD.String(string(r)))...)
	}
	classes := make([]uint8, len(runes))
	for i, r := range runes {
		if r >= utf8.RuneSelf {
			classes[i] = combiningClass(r) // ASCII is all class 0
		}
	}
	reorder(runes, classes)
	return compose(runes, classes)
}

// reorder puts each run of non-starters in canonical order: a stable sort by
// combining class.
func reorder(runes []rune, classes []uint8) {
	for i := 0; i < len(runes); {
		if classes[i] == 0 {
			i++
			continue
		}
		j := i
		for j < len(runes) && classes[j] != 0 {
			j++
		}
		// insertion sort: stable, and runs are short
		for k := i + 1; k < j; k++ {
			for m := k; m > i && classes[m-1] > classes[m]; m-- {
				runes[m-1], runes[m] = runes[m], runes[m-1]
				classes[m-1], classes[m] = classes[m], classes[m-1]
			}
		}
		i = j
	}
}

// compose is canonical composition (UAX #15) of decomposed, canonically
// ordered text.
func compose(runes []rune, classes []uint8) string {
	out := make([]rune, 0, len(runes))
	starter := -1 // index in out of the last starter
	last := -1    // combining class of the last character after it; -1 for none
	for i, r := range runes {
		ccc := int(classes[i])
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

func combiningClass(r rune) uint8 {
	var b [utf8.UTFMax]byte
	n := utf8.EncodeRune(b[:], r)
	return norm.NFD.Properties(b[:n]).CCC()
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
