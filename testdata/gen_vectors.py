# An independent implementation of the spec in the README, which generates
# vectors.json, so the Go code is checked against the spec and not against
# itself. Regenerate with: python3 testdata/gen_vectors.py testdata/vectors.json
import hashlib, json, re, sys, unicodedata

WS = "\t\n\v\f\r \u0085                 　"

def normalize(s):
	s = unicodedata.normalize("NFC", s)
	return " ".join(p for p in re.split("[" + re.escape(WS) + "]+", s) if p)

def h(*fields):
	enc = b"".join(str(len(f.encode())).encode() + b":" + f.encode() + b"," for f in fields)
	return "v2:" + hashlib.sha256(enc).hexdigest()

def tag_child(owner, parent, name): return h("tag", owner, parent, normalize(name))
def tag(owner, *path):
	x = ""
	for n in path: x = tag_child(owner, x, n)
	return x
def question(owner, scope, prompt, answer): return h("question", owner, scope, normalize(prompt), normalize(answer))
def passage(owner, taghash, title): return h("passage", owner, taghash, normalize(title))
def v1(s): return hashlib.sha256(s.encode()).hexdigest()

owner = "user-1"
cells = ["Library", "Science", "Biology", "Cells"]
topic = tag(owner, "Biology", "Unit 1", "Vocabulary")
cases = [
	("a public root", "Tag", ["", "Library"]),
	("a public tag four deep", "Tag", [""] + cells),
	("the same last two names under another parent", "Tag", ["", "Library", "Health", "Biology", "Cells"]),
	("a user's topic", "Tag", [owner, "Biology", "Unit 1", "Vocabulary"]),
	("the same names in another of the user's folders", "Tag", [owner, "Chemistry", "Unit 1", "Vocabulary"]),
	("the same path, public", "Tag", ["", "Biology", "Unit 1", "Vocabulary"]),
	("whitespace is trimmed and collapsed", "Tag", ["", "  Biology", "Unit \t  1 ", "Vocabulary\n"]),
	("a no-break space counts as whitespace", "Tag", ["", "Biology", "Unit 1", "Vocabulary"]),
	("case is kept", "Tag", ["", "biology", "unit 1", "vocabulary"]),
	("an accent typed as one character", "Tag", ["", "Café"]),
	("an accent typed as a combining mark (NFC makes it the same)", "Tag", ["", "Café"]),
	("names that run together differently: AB, C", "Tag", ["", "AB", "C"]),
	("names that run together differently: A, BC", "Tag", ["", "A", "BC"]),
	("symbols and non-Latin text", "Tag", ["", "Brand® Courses", "Course № 1", "心理学 — 概要"]),
	("a root whose name is empty after normalizing", "Tag", ["", "   "]),
	("no names", "Tag", [""]),
	("a child of the four-deep tag, from its parent's hash", "TagChild", ["", tag("", *cells), "Membranes"]),
	("a public question", "Question", ["", "", "What is a bowline?", "A loop"]),
	("the same question, whitespace differing", "Question", ["", "", "What is  a bowline? ", " A loop"]),
	("a question whose prompt and answer run together differently", "Question", ["", "", "What is a bowline?A", " loop"]),
	("a user's question, scoped to their topic", "Question", [owner, topic, "What is a bowline?", "A loop"]),
	("a passage", "Passage", ["", tag("", *cells), "On Cells"]),
	("v1: a public tag", "V1Tag", ["", "Biology", "Cells"]),
	("v1: a public root", "V1Tag", ["", "", "Library"]),
	("v1: a user's tag", "V1Tag", [owner, "Unit 1", "Vocabulary"]),
	("v1: a public question", "V1Question", ["What is a bowline?", "A loop"]),
	("v1: a user's question", "V1UserQuestion", [owner, "cmtopicid0000000000000000", "What is a bowline?", "A loop"]),
	("v1: a passage", "V1Passage", ["On Cells"]),
]

def compute(fn, args):
	if fn == "Tag":
		return tag(args[0], *args[1:]) if len(args) > 1 else ""
	return {
		"TagChild": lambda: tag_child(*args),
		"Question": lambda: question(*args),
		"Passage": lambda: passage(*args),
		"V1Tag": lambda: v1(args[0] + args[1] + args[2]),
		"V1Question": lambda: v1(args[0] + args[1]),
		"V1UserQuestion": lambda: v1("".join(args)),
		"V1Passage": lambda: v1(args[0]),
	}[fn]()

out = [{"name": n, "func": fn, "args": a, "hash": compute(fn, a)} for n, fn, a in cases]
json.dump(out, open(sys.argv[1], "w"), indent="\t", ensure_ascii=False)
open(sys.argv[1], "a").write("\n")
print(len(out), "vectors")
