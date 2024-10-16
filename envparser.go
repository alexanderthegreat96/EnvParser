package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type EnvData struct {
	FilePath    string
	EnvContents map[interface{}]interface{}
	EnvError    error
}

func NewEnvParser(params ...interface{}) *EnvData {
	env := &EnvData{}
	env.EnvParser(params...)
	return env
}

func (env *EnvData) EnvParser(params ...interface{}) {
	var envFilename string
	var useRootPath bool
	var envFileNames []string

	envFilename = ".env"
	useRootPath = true

	if len(params) > 0 {
		if fileName, ok := params[0].(string); ok {
			envFilename = fileName
		}
	}
	if len(params) > 1 {
		if rootPath, ok := params[1].(bool); ok {
			useRootPath = rootPath
		}
	}

	if len(params) > 2 {
		if fileNames, ok := params[2].([]string); ok {
			envFileNames = fileNames
		}
	}

	if len(envFileNames) > 0 {
		for _, fileName := range envFileNames {
			env.parse(fileName, useRootPath)
		}
	}

	env.parse(envFilename, useRootPath)
}

func (env *EnvData) GetError() string {
	return env.EnvError.Error()
}

func (env *EnvData) GetVars() map[interface{}]interface{} {
	// made a copy so that using GetVars and GetValue at the same time
	// does not crash the program
	// this ie because this method used to convert everything by default
	// disabling on-demand conversions

	vars := make(map[interface{}]interface{})
	for key, value := range env.EnvContents {
		vars[key] = value
	}

	if len(vars) == 0 {
		return vars
	}

	for key, value := range vars {
		conv, err := convertToString(value)
		if err != nil {
			env.EnvError = fmt.Errorf("failed to convert value to string for key %v: %w", key, err)
			continue
		}

		convertedValue, err := env.convertInputToType(conv)
		if err != nil {
			env.EnvError = fmt.Errorf("failed to convert input to type for key %v: %w", key, err)
			continue
		}
		vars[key] = convertedValue
	}

	return vars
}

func (env *EnvData) GetEncryptedValue(which, kind string, defaultValue interface{}, decryptionKey string) (interface{}, error) {
	if env.EnvError != nil {
		return nil, env.EnvError
	}

	value, exists := env.EnvContents[which]
	if !exists {
		value = defaultValue
	}

	if !isHashed(value) {
		return nil, fmt.Errorf("value %v is not an encrypted value", value)
	}

	decryptedValue, err := env.decryptValue(value.(string), decryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt value: %w", err)
	}

	if kind != "" && isAllowedType(kind) {
		convertedValue, err := convertToSpecificType(decryptedValue, kind)
		if err != nil {
			return nil, fmt.Errorf("failed to convert decrypted value to type %s: %w", kind, err)
		}
		return convertedValue, nil
	}

	convertedValue, err := env.convertInputToType(fmt.Sprintf("%v", decryptedValue))
	if err != nil {
		return nil, fmt.Errorf("failed to convert decrypted value: %w", err)
	}

	return convertedValue, nil
}

func (env *EnvData) GetValue(which, kind string, defaultValue interface{}) (interface{}, error) {
	if env.EnvError != nil {
		return nil, env.EnvError
	}

	value, exists := env.EnvContents[which]
	if !exists {
		value = defaultValue
	}

	if kind != "" && isAllowedType(kind) {
		convertedValue, err := convertToSpecificType(fmt.Sprintf("%v", value), kind)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value to type %s: %w", kind, err)
		}
		return convertedValue, nil
	}

	convertedValue, err := env.convertInputToType(fmt.Sprintf("%v", value))
	if err != nil {
		return nil, fmt.Errorf("failed to convert value: %w", err)
	}

	return convertedValue, nil
}

func findRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	markerFiles := []string{"go.mod", ".git", ".project-root", ".root"}

	for {
		for _, marker := range markerFiles {
			if _, err := os.Stat(filepath.Join(currentDir, marker)); err == nil {
				return currentDir, nil
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("project root not found")
		}

		currentDir = parentDir
	}
}

func (env *EnvData) parse(fileName string, useRoothPath bool) {
	rootPath, err := findRoot()
	if err != nil {
		env.EnvError = fmt.Errorf("failed to find project root: %w", err)
		return
	}

	env.FilePath = filepath.Join(rootPath, fileName)
	if !useRoothPath {
		env.FilePath = fileName
	}

	if _, err := os.Stat(env.FilePath); os.IsNotExist(err) {
		env.EnvError = fmt.Errorf("env file does not exist: %w", err)
		return
	} else if err != nil {
		env.EnvError = fmt.Errorf("error checking env file: %w", err)
		return
	}

	file, err := os.Open(env.FilePath)
	if err != nil {
		env.EnvError = fmt.Errorf("failed to open file: %w", err)
		return
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	if err := scanner.Err(); err != nil {
		env.EnvError = fmt.Errorf("error while reading file: %w", err)
		return
	}

	if env.EnvContents == nil {
		env.EnvContents = make(map[interface{}]interface{})
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)

		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = value[1 : len(value)-1]
			value = strings.ReplaceAll(value, `\"`, `"`)
		}

		if _, exists := env.EnvContents[key]; !exists {
			env.EnvContents[key] = env.substituteVariables(value, env.EnvContents)
		}

	}
}

func (env *EnvData) convertInputToType(s string) (interface{}, error) {
	if isInteger(s) {
		return strconv.Atoi(s)
	} else if isFloat(s) {
		return strconv.ParseFloat(s, 64)
	} else if isBoolean(s) {
		return strings.ToLower(s) == "true" || strings.ToLower(s) == "false", nil
	} else if isList(s) {
		var list []interface{}

		s = strings.TrimSpace(s[1 : len(s)-1])
		elements := strings.Split(s, ",")
		for _, elem := range elements {
			list = append(list, strings.TrimSpace(elem))
		}
		return list, nil

	} else if isTuple(s) {
		var tuple []interface{}

		s = strings.TrimSpace(s[1 : len(s)-1])
		elements := strings.Split(s, ",")
		for _, elem := range elements {
			tuple = append(tuple, strings.TrimSpace(elem))
		}
		return tuple, nil

	} else if isDict(s) {
		return convertStringToMap(s)
	} else if isJSON(s) {
		var result map[interface{}]interface{}
		if err := json.Unmarshal([]byte(s), &result); err != nil {
			return nil, fmt.Errorf("unable to convert json to map for: %s Error: %w", s, err)
		}

		return result, nil
	} else {
		return s, nil
	}
}

func convertToSpecificType(what, in string) (interface{}, error) {
	foundType := isAllowedType(strings.ToLower(in))
	if !foundType {
		return nil, fmt.Errorf("error: you are attempting to convert: %s in %s, which is not a VALID type", what, in)
	}

	var converted interface{}
	var convertedErr error

	switch strings.ToLower(in) {
	case "str", "string":
		converted = what

	case "bool", "boolean":
		if isBoolean(what) {
			converted = strings.ToLower(what) == "true"
		} else {
			convertedErr = fmt.Errorf("unable to convert value %s to %s", what, in)
		}

	case "float":
		if isFloat(what) {
			converted, convertedErr = strconv.ParseFloat(what, 64)
		} else {
			convertedErr = fmt.Errorf("unable to convert value %s to %s", what, in)
		}

	case "int", "integer":
		if isInteger(what) {
			converted, convertedErr = strconv.Atoi(what)
		} else {
			convertedErr = fmt.Errorf("unable to convert value %s to %s", what, in)
		}
	case "list", "array", "tuple":
		if isList(what) || isTuple(what) {
			var list []interface{}
			trimmed := strings.TrimSpace(what[1 : len(what)-1])
			elements := strings.Split(trimmed, ",")
			for _, elem := range elements {
				list = append(list, strings.TrimSpace(elem))
			}
			converted = list
		} else {
			convertedErr = fmt.Errorf("unable to convert value %s to %s", what, in)
		}

	case "dict", "map", "json":
		if isDict(what) || isJSON(what) {
			converted, convertedErr = convertStringToMap(what)
		} else {
			convertedErr = fmt.Errorf("unable to convert value %s to %s", what, in)
		}

	default:
		converted = what
	}

	return converted, convertedErr
}

func isJSON(myjson any) bool {
	myjsonStr := fmt.Sprintf("%v", myjson)
	var js json.RawMessage
	return json.Unmarshal([]byte(myjsonStr), &js) == nil
}

func isInteger(s any) bool {
	str := fmt.Sprintf("%v", s)
	pattern := `^[+-]?\d+$`
	matched, _ := regexp.MatchString(pattern, str)
	return matched
}

func isFloat(s any) bool {
	str := fmt.Sprintf("%v", s)
	pattern := `^[+-]?\d+(\.\d+)?$`
	matched, _ := regexp.MatchString(pattern, str)
	return matched
}

func isBoolean(s any) bool {
	str := fmt.Sprintf("%v", s)
	return str == "true" || str == "false" || str == "True" || str == "False"
}

func isList(value any) bool {
	str := fmt.Sprintf("%v", value)
	str = strings.TrimSpace(str)
	return strings.HasPrefix(str, "[") && strings.HasSuffix(str, "]")
}

func isDict(value any) bool {
	str := fmt.Sprintf("%v", value)
	str = strings.TrimSpace(str)
	return strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}")
}

func isTuple(value any) bool {
	str := fmt.Sprintf("%v", value)
	str = strings.TrimSpace(str)
	return strings.HasPrefix(str, "(") && strings.HasSuffix(str, ")")
}

func isHashed(value any) bool {
	str := fmt.Sprintf("%v", value)
	str = strings.TrimSpace(str)
	return (strings.HasPrefix(str, "enc(") || strings.HasPrefix(str, "ENC(")) && strings.HasSuffix(str, ")")
}

func convertStringToMap(s string) (map[string]interface{}, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "'") && !strings.Contains(s, "\"") {
		s = strings.ReplaceAll(s, "'", "\"")
	}

	if !(strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) {
		return nil, errors.New("invalid map format")
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal string to map: %w", err)
	}

	return result, nil
}

func convertToString(input interface{}) (string, error) {
	str, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("unable to convert %v to string", input)
	}
	return str, nil
}

func isAllowedType(kind string) bool {
	allowedTypes := []string{
		"str", "string", "bool", "boolean",
		"float", "int", "integer",
		"list", "array", "tuple",
		"dict", "map", "json",
	}
	for _, allowed := range allowedTypes {
		if kind == allowed {
			return true
		}
	}
	return false
}

// funky function that
// uses global env variables
// to create variable subsitution
// in other words
// declare stuff like: {$TEST_VAR}/somehost.com
// and ${TEST_VAR} is replaced with it's equivalent
// found in a different env_file or global env variable
func (env *EnvData) substituteVariables(value string, vars map[interface{}]interface{}) string {
	pattern := regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)
	return pattern.ReplaceAllStringFunc(value, func(match string) string {
		varName := strings.Trim(match, "${}")
		if val, exists := vars[varName]; exists {
			return fmt.Sprintf("%v", val)
		}
		if sysVal, exists := os.LookupEnv(varName); exists {
			return sysVal
		}
		return match
	})
}

// can ether use AES or BASE64 encrypted strings
func (env *EnvData) decryptValue(encryptedValue string, key string) (string, error) {
	var decryptedValue []byte
	var err error

	data := strings.TrimPrefix(encryptedValue, "ENC(")
	data = strings.TrimPrefix(data, "enc(")
	data = strings.TrimSuffix(data, ")")

	if key == "" {
		decryptedValue, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 hash: %w", err)
		}

	} else {
		decryptedValue, err = decryptAES([]byte(encryptedValue), key)
		if err != nil {
			return "", fmt.Errorf("failed to decode AES encrypted value: %w", err)
		}
	}

	return string(decryptedValue), err
}

func decryptAES(encryptedData []byte, key string) ([]byte, error) {
	encryptedData, err := base64.StdEncoding.DecodeString(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 hash: %w", err)
	}

	if len(encryptedData) < aes.BlockSize {
		return nil, fmt.Errorf("encrypted data is too short for AES")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	iv := encryptedData[:aes.BlockSize]
	encryptedData = encryptedData[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	decrypted := make([]byte, len(encryptedData))
	stream.XORKeyStream(decrypted, encryptedData)

	return decrypted, nil
}
