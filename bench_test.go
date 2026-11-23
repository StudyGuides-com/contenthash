package contenthash

import "testing"

var (
	benchASCII = "Ohm's Law (Basic Relation of Voltage Current and Resistance Applied to Set and Control Parameters)"
	benchMixed = "Ohm’s Law (Basic Relation of Voltage Current and Resistance Applied to Set and Control Parameters)"
)

func BenchmarkV2NormalizeASCII(b *testing.B) {
	for i := 0; i < b.N; i++ {
		V2Normalize(benchASCII)
	}
}

func BenchmarkV2NormalizeMixed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		V2Normalize(benchMixed)
	}
}

func BenchmarkV2TagChild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		V2TagChild("", "v2:0123", benchASCII)
	}
}
