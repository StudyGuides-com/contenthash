package contenthash_test

import (
	"fmt"

	"github.com/studyguides-com/contenthash"
)

func ExampleV2Tag() {
	cells := contenthash.V2Tag("", []string{"Science", "Biology", "Cells"})
	other := contenthash.V2Tag("", []string{"Health", "Biology", "Cells"})
	same := contenthash.V2Tag("", []string{"science", "BIOLOGY", " Cells "})
	fmt.Println(contenthash.Version(cells))
	fmt.Println(cells == other)
	fmt.Println(cells == same)
	// Output:
	// 2
	// false
	// true
}

func ExampleV2TagChild() {
	root := contenthash.V2TagChild("", "", "Science")
	biology := contenthash.V2TagChild("", root, "Biology")
	cells := contenthash.V2TagChild("", biology, "Cells")
	fmt.Println(cells == contenthash.V2Tag("", []string{"Science", "Biology", "Cells"}))
	// Output: true
}

func ExampleV2Question() {
	shared := contenthash.V2Question("", "", "What is a cell?", "The basic unit of life")
	again := contenthash.V2Question("", "", " what is a  CELL? ", "The basic unit of life")
	scoped := contenthash.V2Question("", contenthash.V2Tag("", []string{"Science", "Biology"}), "What is a cell?", "The basic unit of life")
	fmt.Println(shared == again)
	fmt.Println(shared == scoped)
	// Output:
	// true
	// false
}

func ExampleV2Normalize() {
	fmt.Printf("%q\n", contenthash.V2Normalize("  Newton\u2019s \t Laws\n"))
	fmt.Printf("%q\n", contenthash.V2Normalize("\u200bUnit\u2013Acquired"))
	fmt.Printf("%q\n", contenthash.V2Normalize("Stra\u00dfe"))
	// Output:
	// "newton's laws"
	// "unit-acquired"
	// "strasse"
}

func ExampleVersion() {
	fmt.Println(contenthash.Version(contenthash.V2Tag("", []string{"Science"})))
	fmt.Println(contenthash.Version(contenthash.V1Tag("", "", "Science")))
	fmt.Println(contenthash.Version("not a hash"))
	// Output:
	// 2
	// 1
	// 0
}
