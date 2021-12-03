package file

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Manifest struct {
	Name   string `yaml:"name"`
	Heroku struct {
		App string   `yaml:"app"`
		Env []string `yaml:"env"`
	} `yaml:"heroku"`
}

func GetManifest(path string) (Manifest, error) {
	var m Manifest
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return m, fmt.Errorf("reading manifest yaml file: %s ", err)
	}
	err = yaml.Unmarshal(yamlFile, &m)
	if err != nil {
		return m, fmt.Errorf("unmarshalling manifest yaml file: %s ", err)
	}
	return m, nil
}
