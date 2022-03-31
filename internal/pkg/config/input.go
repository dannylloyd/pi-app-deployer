package config

import (
	"fmt"
	"strings"
)

type EnvVarFlags struct {
	Map map[string]string
}

func (i *EnvVarFlags) String() string {
	vals := []string{}
	for k, v := range i.Map {
		vals = append(vals, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(vals, ",")
}

func (i *EnvVarFlags) Set(value string) error {
	kv := strings.Split(value, "=")
	if len(kv) != 2 {
		return fmt.Errorf("flag was not properly passed, tried to split flag on '=' but got: '%s'. Original flag used was '%s'", kv, value)
	}
	if i.Map == nil {
		i.Map = make(map[string]string)
	}
	i.Map[kv[0]] = kv[1]
	return nil
}

func (i *EnvVarFlags) Type() string {
	return "TODO"
}
