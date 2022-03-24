package config

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
)

func ValidateEnvVars(m manifest.Manifest, cfg Config) error {
	if len(m.Env) == 0 && len(cfg.EnvVars) == 0 {
		return nil
	}
	keys := make([]string, len(cfg.EnvVars))

	i := 0
	for k := range cfg.EnvVars {
		keys[i] = k
		i++
	}

	sort.Strings(keys)
	sort.Strings(m.Env)
	if !reflect.DeepEqual(keys, m.Env) {
		return fmt.Errorf("manifest defined env vars should exactly match env vars configured in agent install command. Manifest vars: %s, Config var keys: %s", m.Env, keys)
	}

	return nil
}
