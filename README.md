[![build](https://github.com/linkdata/jq/actions/workflows/build.yml/badge.svg)](https://github.com/linkdata/jq/actions/workflows/build.yml)
[![coverage](https://github.com/linkdata/jq/blob/badges/main/badge.svg)](https://github.com/linkdata/jq/blob/badges/main/coverage.html)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/jq)](https://goreportcard.com/report/github.com/linkdata/jq)
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