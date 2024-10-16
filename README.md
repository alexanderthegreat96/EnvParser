# EnvParser

`EnvParser` is a Go package designed for parsing environment variables from `.env` files. It provides functionalities to load, convert, and retrieve environment variable values, including support for encrypted variables. This package is useful for managing configuration settings in Go applications.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Functions](#functions)
  - [NewEnvParser](#newenvparser)
  - [EnvParser](#envparser)
  - [GetVars](#getvars)
  - [GetValue](#getvalue)
  - [GetEncryptedValue](#getencryptedvalue)
- [Error Handling](#error-handling)
- [Variable Substitution](#variable-substitution)
- [Encryption and Decryption](#encryption-and-decryption)
- [License](#license)

## Installation

To use this package, include it in your Go module. You can install it via `go get`:

```bash
go get github.com/alexanderthegreat96/envparser

```
## Usage
Simply initialize the function and then provide it 
```go
package main

import (
    "fmt"
    "your-module-path/envparser" // replace with actual module path
)

func main() {
    // By default, it will use the .env in the root folter
    // You may specify a different file by providing it
    // if your file is not in the project root, set the second argument to false
    // third argument allows you to specify additional env files to be parsed
    parser := envparser.NewEnvParser()
    
    // Get a variable
    // first argument is variable name
    // second argument type to be converted to
    // third is the default value
    value, err := parser.GetValue("YOUR_VAR", "string", "default_value")
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("Value:", value)
    }
}
```
