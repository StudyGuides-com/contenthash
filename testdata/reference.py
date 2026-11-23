# The reference implementation of the spec in the README: an independent
# implementation in Python, which gen_vectors.py and differential.py check the
# Go code against. It derives its character sets from Python's Unicode data
# rather than copying the Go tables, and needs Unicode 15.0 (Python 3.12).
import hashlib, sys, unicodedata

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
