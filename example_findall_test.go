package typedregexp

import "fmt"

const data = `
The Beatles - Abbey Road
Aidan Knight - Each Other
The Dø - Shake Shook Shaken
`

type Album struct {
	Artist string
	Title  string
}

func ExampleTypedRegexp_FindAll() {
	parser := MustCompile("(?m)^{{.Artist}} - {{.Title}}$", Album{
		Artist: `.+`,
		Title:  `.+`,
	})

	albums := make([]Album, 20)
	n := parser.FindAll(data, albums)
	for _, album := range albums[:n] {
		fmt.Println(album)
	}

	// Output:
	// {The Beatles Abbey Road}
	// {Aidan Knight Each Other}
	// {The Dø Shake Shook Shaken}
}
