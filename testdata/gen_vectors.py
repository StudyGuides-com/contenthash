# An independent implementation of the spec in the README, which generates
# vectors.json, so the Go code is checked against the spec and not against
# itself. It derives its character sets from Python's Unicode data rather than
# copying the Go tables, and needs Unicode 15.0 (Python 3.12).
#
#   python3.12 testdata/gen_vectors.py testdata/vectors.json
import hashlib, json, sys, unicodedata

if unicodedata.unidata_version != "15.0.0":
	sys.exit("needs Unicode 15.0.0 (Python 3.12); this Python has %s" % unicodedata.unidata_version)

WHITE_SPACE = set("\u0009\u000a\u000b\u000c\u000d \u0085\u00a0\u1680\u2028\u2029\u202f\u205f\u3000") | {chr(c) for c in range(0x2000, 0x200B)}
# Default_Ignorable_Code_Point as DerivedCoreProperties.txt derives it, from
# Python's category Cf and the PropList.txt 15.0 sets written out here.
OTHER_DEFAULT_IGNORABLE = [(0x034F, 0x034F), (0x115F, 0x1160), (0x17B4, 0x17B5), (0x2065, 0x2065), (0x3164, 0x3164), (0xFFA0, 0xFFA0), (0xFFF0, 0xFFF8), (0xE0000, 0xE0000), (0xE0002, 0xE001F), (0xE0080, 0xE00FF), (0xE01F0, 0xE0FFF)]
VARIATION_SELECTOR = [(0x180B, 0x180D), (0x180F, 0x180F), (0xFE00, 0xFE0F), (0xE0100, 0xE01EF)]
PREPENDED_CONCATENATION_MARK = [(0x0600, 0x0605), (0x06DD, 0x06DD), (0x070F, 0x070F), (0x0890, 0x0891), (0x08E2, 0x08E2), (0x110BD, 0x110BD), (0x110CD, 0x110CD)]
def within(c, ranges): return any(a <= ord(c) <= b for a, b in ranges)
def default_ignorable(c):
	if not (within(c, OTHER_DEFAULT_IGNORABLE) or unicodedata.category(c) == "Cf" or within(c, VARIATION_SELECTOR)):
		return False
	return not (c in WHITE_SPACE or within(c, [(0xFFF9, 0xFFFB), (0x13430, 0x1343F)]) or within(c, PREPENDED_CONCATENATION_MARK))
JOINERS = "\u200c\u200d"
def ignored(c):
	return default_ignorable(c) and c not in JOINERS and not 0xE0020 <= ord(c) <= 0xE007F
DOUBLE_PRIMES = {"\u2033": '"', "\u2036": '"'}
PUNCTUATION = {
	"\u2018": "'", "\u2019": "'", "\u201a": "'", "\u201b": "'", "\u2032": "'", "\u2035": "'", "\u02bc": "'",
	"\u201c": '"', "\u201d": '"', "\u201e": '"', "\u201f": '"',
	"\u2010": "-", "\u2011": "-", "\u2012": "-", "\u2013": "-", "\u2014": "-", "\u2015": "-",
	"\u2212": "-", "\u2e3a": "-", "\u2e3b": "-",
}

def is_space(c):
	return c in WHITE_SPACE or unicodedata.category(c) == "Cc"

def normalize(s):
	s = "".join(DOUBLE_PRIMES.get(c, c) for c in s if not ignored(c))
	s = unicodedata.normalize("NFKC", unicodedata.normalize("NFKC", s).casefold())
	words, word = [], []
	for c in s:
		if is_space(c):
			if word:
				words.append("".join(word))
				word = []
		else:
			word.append(PUNCTUATION.get(c, c))
	if word:
		words.append("".join(word))
	return " ".join(words).strip(" " + JOINERS)

def h(*fields):
	enc = b"".join(str(len(f.encode())).encode() + b":" + f.encode() + b"," for f in fields)
	return "v2:" + hashlib.sha256(enc).hexdigest()

def tag_child(owner, parent, name): return h("tag", owner, parent, normalize(name))
def tag(owner, path):
	x = ""
	for n in path: x = tag_child(owner, x, n)
	return x
def question(owner, scope, prompt, answer): return h("question", owner, scope, normalize(prompt), normalize(answer))
def passage(owner, taghash, title): return h("passage", owner, taghash, normalize(title))
def v1(s): return hashlib.sha256(s.encode()).hexdigest()

ALL_IGNORED = "".join(chr(c) for c in range(sys.maxunicode + 1) if ignored(chr(c)))
owner = "user-1"
cells = ["Library", "Science", "Biology", "Cells"]
topic = tag(owner, ["Biology", "Unit 1", "Vocabulary"])

# (name, func, args, same_as). A Tag's args are the owner and the path. same_as
# names an earlier case the inputs must hash the same as; every other pair of
# cases must hash differently.
cases = [
	# paths
	("a public root", "Tag", ["", ["Library"]], ""),
	("a public tag four deep", "Tag", ["", cells], ""),
	("the same last two names under another parent", "Tag", ["", ["Library", "Health", "Biology", "Cells"]], ""),
	("a user's topic", "Tag", [owner, ["Biology", "Unit 1", "Vocabulary"]], ""),
	("the same names in another of the user's folders", "Tag", [owner, ["Chemistry", "Unit 1", "Vocabulary"]], ""),
	("the same path, public", "Tag", ["", ["Biology", "Unit 1", "Vocabulary"]], ""),
	("names that run together differently: AB, C", "Tag", ["", ["AB", "C"]], ""),
	("names that run together differently: A, BC", "Tag", ["", ["A", "BC"]], ""),
	("a name that looks like the encoding", "Tag", ["", ["3:tag,", "0:,"]], ""),
	("an empty name in the middle of a path", "Tag", ["", ["Science", "", "Cells"]], ""),
	("a root whose name is empty after normalizing", "Tag", ["", ["   "]], ""),
	("a child of the four-deep tag, from its parent's hash", "TagChild", ["", tag("", cells), "Membranes"], ""),
	# whitespace
	("whitespace is trimmed and collapsed", "Tag", ["", ["  Biology", "Unit \t  1 ", "Vocabulary\n"]], "the same path, public"),
	("a newline inside a name is a space", "Tag", ["", ["Biology", "Unit\n1", "Vocabulary"]], "the same path, public"),
	("other whitespace: no-break, ideographic, line separator, NEL, vertical tab, form feed, ogham, narrow no-break", "Tag", ["", ["Biology\u00a0", "Unit\u3000\u2028\u0085\u000b\u000c\u1680\u202f1", "Vocabulary"]], "the same path, public"),
	("control characters are whitespace", "Tag", ["", ["Biology\u0000", "Unit\u001c1", "\u007fVocabulary"]], "the same path, public"),
	# case
	("case is folded", "Tag", ["", ["biology", "UNIT 1", "VoCaBuLaRy"]], "the same path, public"),
	("title case: With", "Tag", ["", ["Genes With Introns"]], ""),
	("title case: with", "Tag", ["", ["Genes with Introns"]], "title case: With"),
	("full case folding: sharp s", "Tag", ["", ["Stra\u00dfe"]], ""),
	("full case folding: SS", "Tag", ["", ["STRASSE"]], "full case folding: sharp s"),
	# punctuation
	("a curly apostrophe", "Tag", ["", ["Newton\u2019s Laws"]], ""),
	("a straight apostrophe", "Tag", ["", ["Newton's Laws"]], "a curly apostrophe"),
	("curly double quotes", "Tag", ["", ["\u201cBlue\u201d Period"]], ""),
	("straight double quotes", "Tag", ["", ['"Blue" Period']], "curly double quotes"),
	("an en dash", "Tag", ["", ["Unit\u2013Acquired"]], ""),
	("a hyphen", "Tag", ["", ["Unit-Acquired"]], "an en dash"),
	("an em dash, a minus sign, a non-breaking hyphen", "Tag", ["", ["Unit\u2014Acquired", "Unit\u2212Acquired", "Unit\u2011Acquired"]], ""),
	("each of those as a hyphen", "Tag", ["", ["Unit-Acquired", "Unit-Acquired", "Unit-Acquired"]], "an em dash, a minus sign, a non-breaking hyphen"),
	("primes", "Tag", ["", ["2\u2032-Deoxy 5\u2032"]], ""),
	("apostrophes for primes", "Tag", ["", ["2'-Deoxy 5'"]], "primes"),
	("a double prime", "Tag", ["", ["12\u2033 Ruler"]], ""),
	("a double quote for a double prime", "Tag", ["", ['12" Ruler']], "a double prime"),
	("a modifier letter apostrophe", "Tag", ["", ["Newton\u02bcs Laws"]], "a curly apostrophe"),
	# invisible characters
	("leading zero width spaces", "Tag", ["", ["\u200b\u200bToxic Metals"]], ""),
	("no zero width spaces", "Tag", ["", ["Toxic Metals"]], "leading zero width spaces"),
	("a byte order mark, word joiner, soft hyphen and direction mark", "Tag", ["", ["\ufeffSci\u00adence\u2060\u200e"]], ""),
	("none of them", "Tag", ["", ["Science"]], "a byte order mark, word joiner, soft hyphen and direction mark"),
	("every ignored character", "Tag", ["", [ALL_IGNORED[:40] + "Sci" + ALL_IGNORED[40:2000] + "ence" + ALL_IGNORED[2000:]]], "none of them"),
	("a combining grapheme joiner and a Hangul filler", "Tag", ["", ["Sci\u034fence\u3164"]], "none of them"),
	("a visible format mark is kept: an Arabic number sign", "Tag", ["", ["\u0600Science"]], ""),
	("a Mongolian vowel separator is removed, not whitespace", "Tag", ["", ["Sci\u180eence"]], "none of them"),
	("zero width joiner and non-joiner are kept inside a name", "Tag", ["", ["Sci\u200dence\u200cs"]], ""),
	("joiners at the ends are trimmed", "Tag", ["", ["\u200d Science \u200c"]], "none of them"),
	("full-width letters", "Tag", ["", ["\uff33\uff43\uff49\uff45\uff4e\uff43\uff45"]], "none of them"),
	# compatibility and canonical forms
	("an accent typed as one character", "Tag", ["", ["Caf\u00e9"]], ""),
	("an accent typed as a combining mark", "Tag", ["", ["Cafe\u0301"]], "an accent typed as one character"),
	("combining marks in one order", "Tag", ["", ["a\u0308\u0323"]], ""),
	("combining marks in the other order", "Tag", ["", ["a\u0323\u0308"]], "combining marks in one order"),
	("Hangul jamo", "Tag", ["", ["\u1112\u1161\u11ab"]], ""),
	("the Hangul syllable", "Tag", ["", ["\ud55c"]], "Hangul jamo"),
	("a ligature", "Tag", ["", ["\ufb01nance"]], ""),
	("no ligature", "Tag", ["", ["finance"]], "a ligature"),
	("a superscript", "Tag", ["", ["x\u00b2"]], ""),
	("no superscript", "Tag", ["", ["x2"]], "a superscript"),
	("NFKC merges a power with its digits: 10 cubed", "Tag", ["", ["10\u00b3"]], ""),
	("NFKC merges a power with its digits: 103", "Tag", ["", ["103"]], "NFKC merges a power with its digits: 10 cubed"),
	("Cherokee folds to uppercase, as CaseFolding.txt says", "Tag", ["", ["\u13a0\u13f5 \uab70\u13fd"]], ""),
	("the same Cherokee, other cases", "Tag", ["", ["\uab70\u13f5 \u13a0\u13f5"]], "Cherokee folds to uppercase, as CaseFolding.txt says"),
	("a tag character before a combining mark doesn't compose", "Tag", ["", ["\U000e0067\u0301"]], ""),
	("a letter g with that mark composes", "Tag", ["", ["g\u0301"]], ""),
	("a CJK Extension B ideograph before U+0338 doesn't compose", "Tag", ["", ["\U0002003c\u0338"]], ""),
	("a composition above U+FFFF: Kaithi, decomposed", "Tag", ["", ["\U00011099\U000110ba"]], ""),
	("the same Kaithi letter, precomposed", "Tag", ["", ["\U0001109a"]], "a composition above U+FFFF: Kaithi, decomposed"),
	("symbols and non-Latin text", "Tag", ["", ["Brand\u00ae Courses", "Course \u2116 1", "\u5fc3\u7406\u5b66 \u2014 \u6982\u8981"]], ""),
	("an emoji with a variation selector", "Tag", ["", ["Stars \u2b50\ufe0f"]], ""),
	("the emoji without it", "Tag", ["", ["Stars \u2b50"]], "an emoji with a variation selector"),
	("England's flag", "Tag", ["", ["\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f History"]], ""),
	("Scotland's flag is a different tag", "Tag", ["", ["\U0001f3f4\U000e0067\U000e0062\U000e0073\U000e0063\U000e0074\U000e007f History"]], ""),
	("a black flag is a different tag", "Tag", ["", ["\U0001f3f4 History"]], ""),
	# questions and passages
	("a public question", "Question", ["", "", "What is a bowline?", "A loop"], ""),
	("the same question, typed differently", "Question", ["", "", "  what is a BOWLINE?", "a\u00a0loop "], "a public question"),
	("a question whose prompt and answer run together differently", "Question", ["", "", "What is a bowline?A", " loop"], ""),
	("an empty prompt", "Question", ["", "", "", "A loop"], ""),
	("an owner, no scope", "Question", [owner, "", "What is a bowline?", "A loop"], ""),
	("a user's question, scoped to their topic", "Question", [owner, topic, "What is a bowline?", "A loop"], ""),
	("a passage", "Passage", ["", tag("", cells), "On Cells"], ""),
	("a passage with an owner", "Passage", [owner, tag(owner, cells), "On Cells"], ""),
	# owner and hash fields are used as given
	("an owner is not case folded", "Tag", ["USER-1", ["Biology", "Unit 1", "Vocabulary"]], ""),
	("an owner is not normalized", "Tag", ["cafe\u0301", ["Science"]], ""),
	("the same owner, precomposed", "Tag", ["caf\u00e9", ["Science"]], ""),
	# v1
	("v1: a public tag", "V1Tag", ["", "Biology", "Cells"], ""),
	("v1: a public root", "V1Tag", ["", "", "Library"], ""),
	("v1: a user's tag", "V1Tag", [owner, "Unit 1", "Vocabulary"], ""),
	("v1: no normalization", "V1Tag", ["", "Biology", "cells "], ""),
	("v1: a public question", "V1Question", ["What is a bowline?", "A loop"], ""),
	("v1: a user's question", "V1UserQuestion", [owner, "cmtopicid0000000000000000", "What is a bowline?", "A loop"], ""),
	("v1: a passage", "V1Passage", ["On Cells"], ""),
]

def compute(fn, a):
	return {
		"Tag": lambda: tag(a[0], a[1:]),
		"TagChild": lambda: tag_child(*a),
		"Question": lambda: question(*a),
		"Passage": lambda: passage(*a),
		"V1Tag": lambda: v1(a[0] + a[1] + a[2]),
		"V1Question": lambda: v1(a[0] + a[1]),
		"V1UserQuestion": lambda: v1("".join(a)),
		"V1Passage": lambda: v1(a[0]),
	}[fn]()

out, by_name, first_with = [], {}, {}
for name, fn, args, same_as in cases:
	if fn == "Tag":
		args = [args[0]] + args[1]  # flat in the file: the owner, then the path
	v = {"name": name, "func": fn, "args": args, "hash": compute(fn, args)}
	if same_as:
		if v["hash"] != by_name[same_as]["hash"]:
			sys.exit("%r should hash the same as %r" % (name, same_as))
	elif v["hash"] in first_with:
		sys.exit("%r unexpectedly hashes the same as %r" % (name, first_with[v["hash"]]))
	first_with.setdefault(v["hash"], name)
	by_name[name] = v
	out.append(v)

with open(sys.argv[1], "w") as f:
	json.dump(out, f, indent="\t", ensure_ascii=True)
	f.write("\n")
print(len(out), "vectors")
