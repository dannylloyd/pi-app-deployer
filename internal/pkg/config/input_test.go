package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_InputValid(t *testing.T) {
	varFlags := EnvVarFlags{"MY_CONFIG=test", "A_CONFIG=foobar"}
	fmt.Println(varFlags)
	m, err := varFlags.FlagToMap()
	assert.NoError(t, err)
	assert.Equal(t, m["MY_CONFIG"], "test")
	assert.Equal(t, m["A_CONFIG"], "foobar")
}

func Test_InputInvalid(t *testing.T) {
	varFlags := EnvVarFlags{"MY_CONFIG test", "A_CONFIG=foobar"}
	fmt.Println(varFlags)
	_, err := varFlags.FlagToMap()
	expectedErr := "flag was not properly passed, tried to split flag on '=' but got: '[MY_CONFIG test]'. Original flag used was 'MY_CONFIG test'"
	assert.EqualError(t, err, expectedErr)

	varFlags = EnvVarFlags{"MY_CONFIG=test", "A_CONFIG foobar"}
	fmt.Println(varFlags)
	_, err = varFlags.FlagToMap()
	expectedErr = "flag was not properly passed, tried to split flag on '=' but got: '[A_CONFIG foobar]'. Original flag used was 'A_CONFIG foobar'"
	assert.EqualError(t, err, expectedErr)
}
