package contenthash

// The character sets V2Normalize uses, written out so they are fixed to
// Unicode 15.0 and don't change with the Go version.

// formatRanges are the Unicode 15.0 format characters (category Cf) that
// V2Normalize removes: every one except U+200C ZERO WIDTH NON-JOINER and
// U+200D ZERO WIDTH JOINER, which change how text renders.
var formatRanges = [][2]rune{
	{0x00AD, 0x00AD}, // soft hyphen
	{0x0600, 0x0605},
	{0x061C, 0x061C},
	{0x06DD, 0x06DD},
	{0x070F, 0x070F},
	{0x0890, 0x0891},
	{0x08E2, 0x08E2},
	{0x180E, 0x180E},
	{0x200B, 0x200B}, // zero width space
	{0x200E, 0x200F}, // direction marks
	{0x202A, 0x202E}, // direction embeddings and overrides
	{0x2060, 0x2064}, // word joiner, invisible operators
	{0x2066, 0x206F}, // direction isolates, deprecated format characters
	{0xFEFF, 0xFEFF}, // byte order mark
	{0xFFF9, 0xFFFB},
	{0x110BD, 0x110BD},
	{0x110CD, 0x110CD},
	{0x13430, 0x1343F},
	{0x1BCA0, 0x1BCA3},
	{0x1D173, 0x1D17A},
	{0xE0001, 0xE0001},
	{0xE0020, 0xE007F}, // tag characters
}

func isFormat(r rune) bool {
	for _, rg := range formatRanges {
		if r < rg[0] {
			return false
		}
		if r <= rg[1] {
			return true
		}
	}
	return false
}

// isSpace reports whether r is whitespace for V2Normalize: the Unicode
// White_Space property, and the control characters (category Cc).
func isSpace(r rune) bool {
	switch {
	case r <= 0x20, r >= 0x7F && r <= 0xA0:
		return true // controls, space, and U+0085 and U+00A0 among C1
	case r == 0x1680, r >= 0x2000 && r <= 0x200A, r == 0x2028, r == 0x2029, r == 0x202F, r == 0x205F, r == 0x3000:
		return true
	}
	return false
}

// foldCherokee corrects x/text's case folding of Cherokee to Unicode's:
// CaseFolding.txt folds Cherokee to its uppercase letters (U+13A0-13F5), but
// cases.Fold lowers them (U+AB70-ABBF, U+13F8-13FD).
func foldCherokee(r rune) rune {
	switch {
	case r >= 0xAB70 && r <= 0xABBF:
		return r - 0xAB70 + 0x13A0
	case r >= 0x13F8 && r <= 0x13FD:
		return r - 8
	}
	return r
}

// punctuation maps look-alike quotes, primes, hyphens and dashes to ASCII.
var punctuation = map[rune]rune{
	0x2018: '\'', // left single quotation mark
	0x2019: '\'', // right single quotation mark
	0x201A: '\'', // single low-9 quotation mark
	0x201B: '\'', // single high-reversed-9 quotation mark
	0x2032: '\'', // prime
	0x2035: '\'', // reversed prime
	0x201C: '"',  // left double quotation mark
	0x201D: '"',  // right double quotation mark
	0x201E: '"',  // double low-9 quotation mark
	0x201F: '"',  // double high-reversed-9 quotation mark
	0x2010: '-',  // hyphen
	0x2011: '-',  // non-breaking hyphen
	0x2012: '-',  // figure dash
	0x2013: '-',  // en dash
	0x2014: '-',  // em dash
	0x2015: '-',  // horizontal bar
	0x2212: '-',  // minus sign
	0x2E3A: '-',  // two-em dash
	0x2E3B: '-',  // three-em dash
}
