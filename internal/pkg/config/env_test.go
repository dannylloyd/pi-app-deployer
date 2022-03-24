package config

import (
	"testing"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/stretchr/testify/assert"
)

func Test_ValidEnv(t *testing.T) {
	m := manifest.Manifest{
		Env: []string{"MY_CONFIG", "HELLO_CONFIG"},
	}

	v := make(map[string]string)
	v["MY_CONFIG"] = "foobar"
	v["HELLO_CONFIG"] = "testing"

	cfg := Config{
		EnvVars: v,
	}

	err := ValidateEnvVars(m, cfg)
	assert.NoError(t, err)
}

func Test_InvalidEnv(t *testing.T) {
	m := manifest.Manifest{
		Env: []string{"MY_CONFIG", "HELLO_CONFIG"},
	}

	v := make(map[string]string)
	v["MY_CONFIG"] = "foobar"

	cfg := Config{
		EnvVars: v,
	}

	err := ValidateEnvVars(m, cfg)
	assert.EqualError(t, err, "manifest defined env vars should exactly match env vars configured in agent install command. Manifest vars: [HELLO_CONFIG MY_CONFIG], Config var keys: [MY_CONFIG]")

	m = manifest.Manifest{
		Env: []string{"MY_CONFIG"},
	}

	v = make(map[string]string)
	v["MY_CONFIG"] = "foobar"
	v["HELLO_CONFIG"] = "testing"

	cfg = Config{
		EnvVars: v,
	}

	err = ValidateEnvVars(m, cfg)
	assert.EqualError(t, err, "manifest defined env vars should exactly match env vars configured in agent install command. Manifest vars: [MY_CONFIG], Config var keys: [HELLO_CONFIG MY_CONFIG]")

	m = manifest.Manifest{
		Env: []string{},
	}

	v = make(map[string]string)
	v["MY_CONFIG"] = "foobar"
	v["HELLO_CONFIG"] = "testing"

	cfg = Config{
		EnvVars: v,
	}

	err = ValidateEnvVars(m, cfg)
	assert.EqualError(t, err, "manifest defined env vars should exactly match env vars configured in agent install command. Manifest vars: [], Config var keys: [HELLO_CONFIG MY_CONFIG]")

	m = manifest.Manifest{
		Env: []string{"MY_CONFIG"},
	}

	v = make(map[string]string)

	cfg = Config{
		EnvVars: v,
	}

	err = ValidateEnvVars(m, cfg)
	assert.EqualError(t, err, "manifest defined env vars should exactly match env vars configured in agent install command. Manifest vars: [MY_CONFIG], Config var keys: []")
}

func Test_EmptyEnv(t *testing.T) {
	m := manifest.Manifest{}

	v := make(map[string]string)

	cfg := Config{
		EnvVars: v,
	}

	err := ValidateEnvVars(m, cfg)
	assert.NoError(t, err)
}
