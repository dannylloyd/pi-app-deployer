package manifest

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v2"
)

const (
	DefaultRestart         = "on-failure"
	DefaultRestartSec      = 5
	DefaultTimeoutStartSec = 0
)

type Manifest struct {
	Name       string        `yaml:"name"`
	Executable string        `yame:"exectutable"`
	Heroku     Heroku        `yaml:"heroku"`
	Systemd    SystemdConfig `yaml:"systemd"`
}

type Heroku struct {
	App string   `yaml:"app"`
	Env []string `yaml:"env"`
}

// SystemdConfig https://www.freedesktop.org/software/systemd/man/systemd.unit.html
type SystemdConfig struct {
	Unit    SystemdUnit    `yaml:"Unit"`
	Service SystemdService `yaml:"Service"`
}

type SystemdUnit struct {
	Description string   `yaml:"Description"`
	After       []string `yaml:"After"`
	Requires    []string `yaml:"Requires"`
}

type SystemdService struct {
	TimeoutStartSec int    `yaml:"TimeoutStartSec"`
	Restart         string `yaml:"Restart"`
	RestartSec      int    `yaml:"RestartSec"`
}

func defaultSystemdUnitAfter() []string {
	return []string{"systemd-journald.service", "network.target"}
}

func defaultSystemdUnitRequires() []string {
	return []string{"systemd-journald.service"}
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

func (m *Manifest) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Manifest
	if err := unmarshal((*plain)(m)); err != nil {
		return err
	}

	var result error

	if m.Name == "" {
		result = multierror.Append(result, fmt.Errorf("name field is required"))
	}

	if m.Executable == "" {
		result = multierror.Append(result, fmt.Errorf("executable field is required"))
	}

	if m.Heroku.App == "" {
		result = multierror.Append(result, fmt.Errorf("heroku.app field is required"))
	}

	if result != nil {
		return result
	}

	if m.Systemd.Service.Restart == "" {
		m.Systemd.Service.Restart = DefaultRestart
	}

	if m.Systemd.Service.RestartSec == 0 {
		m.Systemd.Service.RestartSec = DefaultRestartSec
	}

	if m.Systemd.Service.TimeoutStartSec == 0 {
		m.Systemd.Service.TimeoutStartSec = DefaultTimeoutStartSec
	}

	if m.Systemd.Unit.After == nil {
		m.Systemd.Unit.After = defaultSystemdUnitAfter()
	}

	if m.Systemd.Unit.Requires == nil {
		m.Systemd.Unit.Requires = defaultSystemdUnitRequires()
	}

	if m.Systemd.Unit.Description == "" {
		m.Systemd.Unit.Description = m.Name
	}

	return nil
}
