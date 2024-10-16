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
    "github.com/alexanderthegreat96/envparser" 
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
## Functions
Below, there is a list with all the available functions.

### NewEnvParser
```go
func NewEnvParser(params ...interface{}) *EnvData
```
Will initialize the parser itself. It accepts the following arguments:
 - environemt file name -> default is .env
 - use root path -> default is true (set to false if your file is somewhere else and not in your project)
 - env files -> you may specify an array of env file names. they will be merged together and you will have access to all of them

```go
parser := envparser.NewEnvParser("my.env", true, []string{"another.env", "another_one.env"})
```

### GetVars
```go
func (env *EnvData) GetVars() map[interface{}]interface{}
```
Will return a map with all the variables found across your specified environment file(s). It will also `auto-convert` them to the `correct type`.

### GetValue
```go
func (env *EnvData) GetValue(which, kind string, defaultValue interface{}) (interface{}, error)
```
This function will grab a value from the file by key, will convert it to the type you specify and if not found, will return the default value specified.
Arguments:
 - key -> your variable name
 - kind -> what do you want this converted to? (check the types below)
 - default value -> if not found, will default to this

Supported types:
- str
- string
- bool
- boolean
- float
- int
- integer
- list
- array
- tuple
- dict
- map
- json

### GetEncryptedValue
```go
func (env *EnvData) GetEncryptedValue(which, kind string, defaultValue interface{}, decryptionKey string) (interface{}, error)
```
Same structure as above, except, you may provide a decryption key. It supports `base64` and `AES`.
Usage:
 - for `base64` hashes, simply call it without providing an encryption key
 - for `aes` encryption, provide the basee64 hash +  the encryption key

### Error Handling
```go
func (env *EnvData) GetError() string
```
Any errors that may have occured can be captured with this.

## Variable Substitution
The package supports variable substitution, allowing you to reference other environment variables within the .env file. For example:
```bash
API_URL=https://${API_HOST}:${API_PORT}/api
```
The values will be replaced with the ones found in other env files.