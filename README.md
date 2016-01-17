# typedregexp [![Build Status](https://travis-ci.org/zach-klippenstein/go-typedregexp.svg?branch=master)](https://travis-ci.org/zach-klippenstein/go-typedregexp) [![GoDoc](https://godoc.org/github.com/zach-klippenstein/go-typedregexp?status.svg)](https://godoc.org/github.com/zach-klippenstein/go-typedregexp)

`typedregexp` matches regular expressions into structs.

```go
import (
	"fmt"
	"log"

	"github.com/zach-klippenstein/go-typedregexp"
)

type Values struct {
	Name string
	Age  string
}

func main() {
	re, _ := typedregexp.Compile("Hi, I'm {{.Name}}. I'm {{.Age}} years old!|I'm {{.Name}}, I'm {{.Age}}.", Values{
		Name: `\w+`,
		Age:  `\d+`,
	})

	var values Values
	if re.Find("Hi, I'm Sam. I'm 20 years old!", &values) {
		fmt.Printf("%+v", values)
	} else {
		fmt.Println("No match.")
	}
}

```

Prints `{Name:Sam Age:20}`.

[See the godoc for more examples.](https://godoc.org/github.com/zach-klippenstein/go-typedregexp#pkg-examples)
