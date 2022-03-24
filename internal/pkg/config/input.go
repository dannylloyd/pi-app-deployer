package config

import (
	"fmt"
	"strings"
)

type EnvVarFlags []string

func (i *EnvVarFlags) String() string {
	return strings.Join(*i, " ")
}

func (i *EnvVarFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *EnvVarFlags) FlagToMap() (map[string]string, error) {
	m := make(map[string]string)
	for _, e := range *i {
		kv := strings.Split(e, "=")
		if len(kv) != 2 {
			return m, fmt.Errorf("flag was not properly passed, tried to split flag on '=' but got: '%s'. Original flag used was '%s'", kv, e)
		}
		m[kv[0]] = kv[1]
	}
	return m, nil
}
