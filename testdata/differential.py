# Compares the Go V2Normalize with the reference implementation (reference.py)
# on every code point, its decomposed, compatibility and case-folded forms,
# characters above U+FFFF before combining marks, long runs of combining
# marks, and random strings of tricky characters; and checks that normalizing
# twice changes nothing, in both. Exits 1 on any difference. Needs Unicode 15.0
# (Python 3.12) and Go.
#
#   python3.12 testdata/differential.py [--random N] [--seed S]
import argparse, importlib.util, json, os, random, subprocess, sys, unicodedata

here = os.path.dirname(os.path.abspath(__file__))
_spec = importlib.util.spec_from_file_location("reference", os.path.join(here, "reference.py"))
reference = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(reference)
normalize = reference.normalize

parser = argparse.ArgumentParser()
parser.add_argument("--random", type=int, default=300000, help="random strings to add")
parser.add_argument("--seed", type=int, default=1)
args = parser.parse_args()
rng = random.Random(args.seed)

def code_points():
	return (chr(c) for c in range(sys.maxunicode + 1) if not 0xD800 <= c <= 0xDFFF)

inputs = []
for c in code_points():
	inputs.append(c)
	for form in (unicodedata.normalize("NFD", c), unicodedata.normalize("NFKD", c), c.casefold()):
		if form != c:
			inputs.append(form)
count_points = len(inputs)

# characters above U+FFFF whose low 16 bits are a letter that composes with a mark
marks = [chr(c) for c in range(0x300, 0x370)] + ["\u3099", "\u309a", "\u0b3e", "\u0cd5", "\u102e", "\u1161", "\u11a8"]
for b in range(0x20, 0x10000):
	base = chr(b)
	composes = [m for m in marks if len(unicodedata.normalize("NFC", base + m)) == 1 and unicodedata.normalize("NFC", base + m) != base + m]
	for plane in range(1, 17):
		for m in composes[:2]:
			inputs.append(chr(plane * 0x10000 + b) + m)
count_supplementary = len(inputs) - count_points

# long runs of combining marks, past the 30 non-starters of UAX #15's
# stream-safe format
run_marks = marks + ["\u0344", "\u0f73", "\u0f75", "\u0f81", "\u1ab0", "\u20d0", "\u0e48"]
run_bases = list("aAeEoOgGnN") + ["\U000e0067", "\U0002003c", "\u1100", "\uac00", "\u0b47", "\U00011099"]
for _ in range(20000):
	s = rng.choice(run_bases) + "".join(rng.choice(run_marks) for _ in range(rng.randint(20, 70)))
	inputs.append(s)

# random strings of characters that exercise every step
alphabet = list(
	" \t\n\u00a0\u3000\u2028\u0085\u000b\u001c\u007f\u0000"  # whitespace and controls
	"\u200b\u200c\u200d\ufeff\u2060\u00ad\u180e\u034f\u3164\ufe0f\ufe0e\U000e0001\U000e0067\U000e007f\u0600"  # ignorables, joiners, tags
	"\u0301\u0308\u0323\u0300\u1112\u1161\u11abaAeEsS"  # combining marks, Hangul jamo, ASCII
	"\u00df\u0130\u0131\u01c5\u01c4\ufb01\u00b2\u00b3\uff21\u212b\u00c5\u00e9\u03a3\u03c3\u03c2\u0345"  # case folding, compatibility
	"\u13a0\uab70\u13f8"  # Cherokee
	"\u2018\u2019\u201c\u201d\u2013\u2014\u2011\u2212\u2032\u2033\u2036\u02bc\u00b4"  # punctuation
	"\u2b50\U0001f3f4\U00011099\U000110ba\U0002003c\u0338"  # emoji, flags, supplementary compositions
)
for _ in range(args.random):
	inputs.append("".join(rng.choice(alphabet) for _ in range(rng.randint(0, 16))))

def run_go(strings):
	proc = subprocess.run(["go", "run", "./testdata/normalize"], cwd=os.path.dirname(here),
		input="".join(json.dumps(s) + "\n" for s in strings), capture_output=True, text=True)
	if proc.returncode != 0:
		sys.exit("the Go normalizer failed: " + proc.stderr)
	return [json.loads(line) for line in proc.stdout.splitlines()]

print("%d inputs: %d code point forms, %d supplementary + mark pairs, 20000 long mark runs, %d random" % (len(inputs), count_points, count_supplementary, args.random))
go = run_go(inputs)
if len(go) != len(inputs):
	sys.exit("the Go normalizer returned %d results for %d inputs" % (len(go), len(inputs)))

differences, py_unstable = [], 0
for s, g in zip(inputs, go):
	p = normalize(s)
	if p != g:
		differences.append((s, g, p))
	if normalize(p) != p:
		py_unstable += 1
go_twice = run_go(go)
go_unstable = sum(1 for once, twice in zip(go, go_twice) if once != twice)

for s, g, p in differences[:10]:
	print("differ: %s  go %s  reference %s" % (ascii(s), ascii(g), ascii(p)))
print("differences: %d   not idempotent: go %d, reference %d" % (len(differences), go_unstable, py_unstable))
sys.exit(1 if differences or go_unstable or py_unstable else 0)
