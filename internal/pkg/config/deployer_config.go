package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

var DeployerConfigFile = fmt.Sprintf("%s/.pi-app-deployer.config.yaml", PiAppDeployerDir)

type DeployerConfig struct {
	HerokuApp string `yaml:"herokuApp"`
	// TODO: should this really just be a manifest?
	AppConfigs map[string]Config `yaml:"appConfigs"`
	Path       string            `yaml:"path,omitempty"`
}

func NewDeployerConfig(path, herokuApp string) (DeployerConfig, error) {
	defaultConfig := DeployerConfig{
		HerokuApp:  herokuApp,
		Path:       path,
		AppConfigs: map[string]Config{},
	}

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		if err.Error() == fmt.Sprintf("open %s: no such file or directory", path) {
			return defaultConfig, nil
		}
		return defaultConfig, fmt.Errorf("reading app configs yaml file: %s", err)
	}

	err = yaml.Unmarshal(yamlFile, &defaultConfig)
	if err != nil {
		return defaultConfig, fmt.Errorf("unmarshalling app configs %s", err)
	}

	return defaultConfig, nil
}

// WriteDeployerConfig will overwrite the whole file.
// It is up to the caller to make sure all contents are there.
func (a *DeployerConfig) WriteDeployerConfig() error {
	out, err := yaml.Marshal(a)
	if err != nil {
		return err
	}

	err = os.WriteFile(a.Path, out, 0644)
	if err != nil {
		return fmt.Errorf("writing app configs: %s", err)
	}
	return nil
}

func (d *DeployerConfig) SetAppConfig(c Config) {
	d.AppConfigs[configToKey(c)] = c
}

func (d *DeployerConfig) ConfigExists(c Config) bool {
	_, ok := d.AppConfigs[configToKey(c)]
	return ok
}

func configToKey(c Config) string {
	return strings.ReplaceAll(fmt.Sprintf("%s_%s", c.RepoName, c.ManifestName), "/", "_")
}
