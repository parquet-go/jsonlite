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
