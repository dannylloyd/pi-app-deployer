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
	Systemd struct {
		Unit    SystemdUnit    `yaml:"Unit"`
		Service SystemdService `yaml:"Service"`
	} `yaml:"systemd"`
}

type SystemdUnit struct {
	Description string `yaml:"Description"`
	After       string `yaml:"After"`
	Requires    string `yaml:"Requires"`
}

type SystemdService struct {
	TimeoutStartSec int    `yaml:"TimeoutStartSec"`
	Restart         string `yaml:"Restart"`
	RestartSec      string `yaml:"RestartSec"`
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
