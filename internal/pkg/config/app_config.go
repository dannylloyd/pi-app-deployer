package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

var AppConfigsFile = fmt.Sprintf("%s/.pi-app-deployer.appconfigs.yaml", PiAppDeployerDir)

type AppConfigs struct {
	Map map[string]Config `yaml:"map"`
}

func GetAppConfigs(path string) (AppConfigs, error) {
	var emptyAppConfigs AppConfigs
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		if err.Error() == fmt.Sprintf("open %s: no such file or directory", path) {
			return AppConfigs{Map: map[string]Config{}}, nil
		}
		return emptyAppConfigs, fmt.Errorf("reading app configs yaml file: %s", err)
	}

	err = yaml.Unmarshal(yamlFile, &emptyAppConfigs)
	if err != nil {
		return emptyAppConfigs, fmt.Errorf("unmarshalling app configs %s", err)
	}

	return emptyAppConfigs, nil
}

// WriteAppConfigs will overwrite the whole file.
// It is up to the caller to make sure all contents are there.
func (a *AppConfigs) WriteAppConfigs(path string) error {
	out, err := yaml.Marshal(a)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, out, 0644)
	if err != nil {
		return fmt.Errorf("writing app configs: %s", err)
	}
	return nil
}

func (a *AppConfigs) SetConfig(c Config) {
	a.Map[configToKey(c)] = c
}

func (a *AppConfigs) ConfigExists(c Config) bool {
	_, ok := a.Map[configToKey(c)]
	return ok
}

func configToKey(c Config) string {
	return strings.ReplaceAll(fmt.Sprintf("%s_%s", c.RepoName, c.ManifestName), "/", "_")
}
