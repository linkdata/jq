[![build](https://github.com/linkdata/jq/actions/workflows/build.yml/badge.svg)](https://github.com/linkdata/jq/actions/workflows/build.yml)
[![coverage](https://github.com/linkdata/jq/blob/coverage/main/badge.svg)](https://html-preview.github.io/?url=https://github.com/linkdata/jq/blob/coverage/main/report.html)
[![Docs](https://godoc.org/github.com/linkdata/jq?status.svg)](https://godoc.org/github.com/linkdata/jq)

# jq

Go JSON structure query path getter/setter

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/linkdata/jq"
)

const rawJson = `{
  "name": "John Doe",
  "age": 30,
  "isStudent": false,
  "hobbies": ["reading", "hiking", "gaming"],
  "address": {
    "street": "123 Main St",
    "city": "Anytown",
    "zip": "12345"
  }
}`

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    string `json:"zip"`
}

type Person struct {
	Name      string   `json:"name"`
	Age       int      `json:"age"`
	IsStudent bool     `json:"isStudent"`
	Hobbies   []string `json:"hobbies"`
	Address   Address  `json:"address"`
}

func main() {
	var person Person
	var err error
	if err = json.Unmarshal([]byte(rawJson), &person); err == nil {
		var firsthobby string
		if firsthobby, err = jq.GetAs[string](&person, "hobbies.0"); err == nil {
			fmt.Println(firsthobby)
			var address Address
			if address, err = jq.GetAs[Address](&person, "address"); err == nil {
				fmt.Println(address.City)
			}
		}
	}
	if err != nil {
		panic(err)
	}
	// Output:
	// reading
	// Anytown
}
```

## Checked updates

`SetChecked` tentatively applies the same operation as `Set`, then calls a
checker against the resulting object. A checker error restores the original
object and is returned unchanged. A checker panic also restores the object
before the panic continues.

The checker runs only when `Set` reports a write. It may inspect or marshal the
tentative object, but it must not mutate it. Callers are responsible for
synchronizing access throughout `SetChecked`, including while the checker runs.
