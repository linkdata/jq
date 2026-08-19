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
An untagged anonymous struct field contributes its promoted fields directly to
the containing struct's namespace; it does not add its Go type name as a path
component. For example, an anonymous `Inner` exposes `value`. Tagging the field
`json:"inner"` replaces that path with `inner.value`; `json:"Inner"` uses
`Inner.value`.

Reachable exported fields promoted through unexported embedded structs are
readable with `Get` and writable with `Set` when addressable, including through
map-to-struct assignments.

An exact `json:"-"` tag excludes a field from path traversal and map-to-struct
assignments.

When an unexported embedded field has an explicit JSON name, `Get` and `Set` can
traverse that component on paths to reachable exported fields. A path ending at
the embedded field returns `ErrPathNotFound`, as does a map-to-struct assignment
to it.

## Assignments

`Set` converts among integer kinds other than `uintptr` and floating-point kinds
using
[Go's numeric conversion rules](https://go.dev/ref/spec#Conversions_between_numeric_types).
These conversions can truncate, wrap, or lose precision and do not report
overflow.

For map inputs assigned to structs, only entries with matching string keys
update fields; all other entries are ignored.

A nil interface value supplied by the map stores the field's zero value. A typed
nil pointer supplied for a pointer or interface field retains its type and must
be assignable to that field. For other fields, it stores the field's zero value
only when its pointed-to type is assignable or supported by `Set`'s
numeric-conversion or map-to-struct rules; otherwise `Set` returns
`ErrTypeMismatch`. `Set` dereferences a non-nil pointer for a non-pointer field,
except that for an interface field it does so only when the pointed-to type
implements the interface.

For an existing struct, unselected fields are retained and `Set` reports no
write if no selected field changes; an appended struct starts from zero.
Existing overlays are shallow: preserved pointers retain identity, and
successful updates to promoted fields reached through embedded pointers are
visible through other aliases.

When an interface contains a struct value, `Set` cannot replace values stored
inline in that struct, including nested struct fields and array elements. It can
update pointees, existing map entries, and existing slice elements reachable
from the struct. Attempts to replace an unaddressable field or array element, or
to grow a slice with an unaddressable header, return `ErrPathNotFound`.

## Change detection

`Set` reports whether it performed a write. For an existing destination value,
`Set` skips an assignable replacement when
[`reflect.DeepEqual`](https://pkg.go.dev/reflect#DeepEqual) reports equality.
This is Go deep equality, not equality of serialized JSON; `Set` does not
marshal values to make this decision.

Consequently, `Set` can report no write and retain an existing pointer, map, or
slice, including its aliasing, when a distinct replacement is deeply equal. The
replacement is not installed. When their referenced data is independent, later
mutations through the replacement are not visible through the retained value.

## Checked updates

`SetChecked` tentatively applies the same operation as `Set`, then calls a
checker against the resulting object. A checker error restores the original
object and is returned unchanged. A checker panic also restores the object
before the panic continues.

The checker runs only when `Set` reports a write. A deeply equal assignable
replacement skipped by `Set` therefore does not invoke it. The checker may
inspect or marshal the tentative object, but it must not mutate it. When the
checker uses `Get`, a path ending at an explicitly named unexported embedded
field returns `ErrPathNotFound`, even after a tentative update beneath it.
Callers are responsible for synchronizing access throughout `SetChecked`,
including while the checker runs.
