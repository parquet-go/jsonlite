# jsonlite

A lightweight JSON parser for Go, optimized for performance through careful memory management.

## Installation

```bash
go get github.com/parquet-go/jsonlite
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/parquet-go/jsonlite"
)

func main() {
    // Parse JSON string
    val, err := jsonlite.Parse(`{"name": "Alice", "age": 30, "active": true}`)
    if err != nil {
        panic(err)
    }

    // Access object fields
    if name := val.Lookup("name"); name != nil {
        fmt.Println("Name:", name.String())
    }

    if age := val.Lookup("age"); age != nil {
        fmt.Println("Age:", age.Int())
    }

    // Serialize back to JSON
    buf := val.Append(nil)
    fmt.Println("JSON:", string(buf))
}
```

## API

### Parsing

- `Parse(data string) (*Value, error)` - Parse a JSON string
- `ParseBytes(data []byte) (*Value, error)` - Parse JSON from a byte slice

### Value Types

The `Kind` type represents JSON value types:

- `Null` - JSON null
- `True` - JSON true
- `False` - JSON false
- `Number` - JSON number
- `String` - JSON string
- `Object` - JSON object
- `Array` - JSON array

### Value Methods

- `Kind() Kind` - Get the type of the value
- `Len() int` - Get length (strings, numbers, arrays, objects)
- `String() string` - Get string value (for strings/numbers returns raw value, for other types returns JSON representation)
- `Int() int64` - Parse as signed integer (numbers only)
- `Uint() uint64` - Parse as unsigned integer (numbers only)
- `Float() float64` - Parse as float (numbers only)
- `Array() []Value` - Get array elements (arrays only)
- `Object() []Field` - Get object fields (objects only)
- `Lookup(key string) *Value` - Find field by key (objects only)
- `Number() json.Number` - Get value as json.Number (numbers only)
- `NumberType() NumberType` - Get number classification (numbers only)
- `Append(buf []byte) []byte` - Serialize value to JSON, appending to buffer

### String Unquoting

- `Unquote(s string) (string, error)` - Remove quotes and process escape sequences

### Number Classification

- `NumberTypeOf(s string) NumberType` - Classify a number string as `Int`, `Uint`, or `Float`

## Safety

All accessor methods include type checks and will panic if called on the wrong value type. Always check `Kind()` before calling type-specific methods.

```go
val, _ := jsonlite.Parse(`[1, 2, 3]`)

// Safe: check kind first
if val.Kind() == jsonlite.Array {
    for _, elem := range val.Array() {
        // ...
    }
}

// Unsafe: will panic if val is not an array
_ = val.Array()
```

## License

See LICENSE file.
