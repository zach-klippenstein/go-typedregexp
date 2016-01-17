package typedregexp

import (
	"fmt"
	"log"
)

type ValuesDefaults struct {
	Name string
	Age  string
}

func ExampleTypedRegexp_Find_defaultValues() {
	re, err := Compile("Hi, I'm {{.Name}}. I'm {{.Age}} years old!|I'm {{.Name}}, I'm {{.Age}}.", ValuesDefaults{
		Name: `\w+`,
		Age:  `\d+`,
	})
	if err != nil {
		log.Fatal(err)
	}

	values := ValuesDefaults{
		Name: "Jane Doe",
		Age: "-1",
	}
	re.Find("I'll ask the questions.", &values)
	fmt.Printf("%+v", values)

	// Output:
	// {Name:Jane Doe Age:-1}
}
