// Command normalize reads JSON strings, one per line, and writes each one's
// V2Normalize as a JSON string, one per line. testdata/differential.py runs
// it to compare the Go code with the reference implementation.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/studyguides-com/contenthash"
)

func main() {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 1<<24)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	enc := json.NewEncoder(out)
	for in.Scan() {
		var s string
		if err := json.Unmarshal(in.Bytes(), &s); err != nil {
			fmt.Fprintln(os.Stderr, "normalize:", err)
			os.Exit(1)
		}
		if err := enc.Encode(contenthash.V2Normalize(s)); err != nil {
			fmt.Fprintln(os.Stderr, "normalize:", err)
			os.Exit(1)
		}
	}
	if err := in.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "normalize:", err)
		os.Exit(1)
	}
}
