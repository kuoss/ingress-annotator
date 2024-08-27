package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustMarshalJSON_ValidStruct(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	input := testStruct{
		Name:  "John Doe",
		Age:   30,
		Email: "john.doe@example.com",
	}

	expected := `{"name":"John Doe","age":30,"email":"john.doe@example.com"}`

	result := MustMarshalJSON(input)
	assert.JSONEq(t, expected, string(result), "The JSON should match the expected value")
}

func TestMustMarshalJSON_InvalidInput(t *testing.T) {
	invalidInput := make(chan int)

	assert.Panics(t, func() {
		MustMarshalJSON(invalidInput)
	}, "The function should panic on invalid input")
}
