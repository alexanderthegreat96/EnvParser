package envparser

import (
	"os"
	"testing"
)

func TestNewEnvParser(t *testing.T) {
	env := NewEnvParser()
	if env.EnvContents == nil {
		t.Error("Expected EnvContents to be initialized")
	}
}

func TestEnvParser_ValidFile(t *testing.T) {
	testFile := ".env.test"
	err := os.WriteFile(testFile, []byte("TEST_VAR=value\nANOTHER_VAR=another_value"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	env := NewEnvParser(testFile)
	env.EnvParser(testFile)

	if env.EnvError != nil {
		t.Errorf("Expected no error but got: %v", env.EnvError)
	}

	if env.EnvContents["TEST_VAR"] != "value" {
		t.Errorf("Expected TEST_VAR to be 'value', got: %v", env.EnvContents["TEST_VAR"])
	}
}

func TestEnvParser_NonExistentFile(t *testing.T) {
	env := NewEnvParser("non_existent.env")
	env.EnvParser()

	if env.EnvError == nil {
		t.Error("Expected an error for non-existent file but got none")
	}
}

func TestGetValue_ExistingVariable(t *testing.T) {
	env := NewEnvParser()
	env.EnvContents = map[interface{}]interface{}{
		"EXISTING_VAR": "some_value",
	}

	value, err := env.GetValue("EXISTING_VAR", "", nil)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
	if value != "some_value" {
		t.Errorf("Expected 'some_value', got: %v", value)
	}
}

func TestGetValue_NonExistingVariable(t *testing.T) {
	env := NewEnvParser()
	env.EnvContents = map[interface{}]interface{}{}

	value, err := env.GetValue("NON_EXISTING_VAR", "", "default_value")
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
	if value != "default_value" {
		t.Errorf("Expected 'default_value', got: %v", value)
	}
}

func TestGetValue_ErrorConversion(t *testing.T) {
	env := NewEnvParser()
	env.EnvContents = map[interface{}]interface{}{
		"INVALID_VAR": "not_an_int",
	}

	_, err := env.GetValue("INVALID_VAR", "int", nil)
	if err == nil {
		t.Error("Expected an error for invalid conversion but got none")
	}
}

func TestSubstituteVariables(t *testing.T) {
	env := NewEnvParser()
	env.EnvContents = map[interface{}]interface{}{
		"TEST_VAR": "substituted_value",
	}

	result := env.substituteVariables("URL is ${TEST_VAR}/some/path", env.EnvContents)
	expected := "URL is substituted_value/some/path"
	if result != expected {
		t.Errorf("Expected '%s', got: '%s'", expected, result)
	}
}

func TestGetBase64EncryptedValue(t *testing.T) {
	env := NewEnvParser()
	encryptedValue := "ENC(YXNkamtuYWtqc2Ric2prYmRma2pzaGRiZg==)"
	decryptionKey := ""

	env.EnvContents = map[interface{}]interface{}{
		"my_encrypted_var": encryptedValue,
	}

	decryptedValue, err := env.GetEncryptedValue("my_encrypted_var", "", "expected_value", decryptionKey)
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	expectedValue := "asdjknakjsdbsjkbdfkjshdbf"
	if decryptedValue != expectedValue {
		t.Errorf("Expected '%s', got: '%v'", expectedValue, decryptedValue)
	}
}
