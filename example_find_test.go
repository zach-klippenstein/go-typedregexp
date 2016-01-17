package typedregexp

import (
	"fmt"
	"log"
)

type Values struct {
	Name string
	Age  string
}

func ExampleTypedRegexp_Find() {
	re, err := Compile("Hi, I'm {{.Name}}. I'm {{.Age}} years old!|I'm {{.Name}}, I'm {{.Age}}.", Values{
		Name: `\w+`,
		Age:  `\d+`,
	})
	if err != nil {
		log.Fatal(err)
	}

	var values Values
	if re.Find("Hi, I'm Sam. I'm 20 years old!", &values) {
		fmt.Printf("%+v", values)
	} else {
		fmt.Println("No match.")
	}

	// Output:
	// {Name:Sam Age:20}
}
