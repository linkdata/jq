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

## Struct fields

Struct path components exactly match names selected by `encoding/json`'s default
field-selection rules, including JSON tag names and unambiguous promoted fields.
Reachable exported fields promoted through unexported embedded structs are
readable with `Get` and writable with `Set`, including through map-to-struct
assignments.

An exact `json:"-"` tag excludes a field from path traversal and map-to-struct
assignments.

When an unexported embedded field has an explicit JSON name, `Get` and `Set` can
traverse that component on paths to reachable exported fields. A path ending at
the embedded field returns `ErrPathNotFound`, as does a map-to-struct assignment
to it.

## Checked updates

`SetChecked` tentatively applies the same operation as `Set`, then calls a
checker against the resulting object. A checker error restores the original
object and is returned unchanged. A checker panic also restores the object
before the panic continues.

The checker runs only when `Set` reports a write. It may inspect or marshal the
tentative object, but it must not mutate it. When the checker uses `Get`, a path
ending at an explicitly named unexported embedded field returns
`ErrPathNotFound`, even after a tentative update beneath it. Callers are
responsible for synchronizing access throughout `SetChecked`, including while
the checker runs.
