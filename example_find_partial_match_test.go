package typedregexp

import (
	"fmt"
	"log"
)

type ValuesPartialMatch struct {
	Name string
	Age  string
}

func ExampleTypedRegexp_Find_partialMatch() {
	re, err := Compile("Hi, I'm {{.Name}}.( I'm {{.Age}} years old!)?", ValuesPartialMatch{
		Name: `\w+`,
		Age:  `\d+`,
	})
	if err != nil {
		log.Fatal(err)
	}

	var values ValuesPartialMatch
	re.Find("Hi, I'm Sam.", &values)
	fmt.Printf("%+v", values)

	// Output:
	// {Name:Sam Age:}
}
