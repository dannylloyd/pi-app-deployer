package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Input(t *testing.T) {
	varFlags := EnvVarFlags{}
	err := varFlags.Set("hello=world")
	assert.NoError(t, err)

	err = varFlags.Set("helloworld")
	assert.EqualError(t, err, "flag was not properly passed, tried to split flag on '=' but got: '[helloworld]'. Original flag used was 'helloworld'")

	err = varFlags.Set("MY_CONFIG=test")
	assert.NoError(t, err)
	assert.Equal(t, varFlags.Map["MY_CONFIG"], "test")

	err = varFlags.Set("A_CONFIG=foobar")
	assert.NoError(t, err)
	assert.Equal(t, varFlags.Map["A_CONFIG"], "foobar")
}
