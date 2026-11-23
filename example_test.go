package contenthash_test

import (
	"fmt"

	"github.com/studyguides-com/contenthash"
)

func ExampleTag() {
	cells := contenthash.Tag("", "Science", "Biology", "Cells")
	other := contenthash.Tag("", "Health", "Biology", "Cells")
	fmt.Println(contenthash.IsV2(cells))
	fmt.Println(cells == other)
	// Output:
	// true
	// false
}

func ExampleTagChild() {
	root := contenthash.TagChild("", "", "Science")
	biology := contenthash.TagChild("", root, "Biology")
	cells := contenthash.TagChild("", biology, "Cells")
	fmt.Println(cells == contenthash.Tag("", "Science", "Biology", "Cells"))
	// Output: true
}

func ExampleQuestion() {
	shared := contenthash.Question("", "", "What is a cell?", "The basic unit of life")
	again := contenthash.Question("", "", " What is a  cell? ", "The basic unit of life")
	scoped := contenthash.Question("", contenthash.Tag("", "Science", "Biology"), "What is a cell?", "The basic unit of life")
	fmt.Println(shared == again)
	fmt.Println(shared == scoped)
	// Output:
	// true
	// false
}

func ExampleNormalize() {
	fmt.Printf("%q\n", contenthash.Normalize("  Unit \t 1\n"))
	fmt.Println(contenthash.Normalize("Café") == "Café")
	// Output:
	// "Unit 1"
	// true
}

func ExampleIsV2() {
	fmt.Println(contenthash.IsV2(contenthash.Tag("", "Science")))
	fmt.Println(contenthash.IsV2(contenthash.V1Tag("", "", "Science")))
	// Output:
	// true
	// false
}
