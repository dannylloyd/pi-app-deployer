package manifest

import (
	"bytes"
	"fmt"
	"io"
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
	// Name is used for the systemd unit file. TODO: add validation for spaces, strings, etc
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

func GetManifest(path, manifestName string) (Manifest, error) {
	var emptyManifest Manifest
	yamlFile, err := ioutil.ReadFile(path)

	if err != nil {
		return emptyManifest, fmt.Errorf("reading manifest yaml file: %s ", err)
	}

	manifests := []Manifest{}
	if err := unmarshalAllManifests(yamlFile, &manifests); err != nil {
		return emptyManifest, err
	}

	err = checkDuplicateManifests(manifests)
	if err != nil {
		return emptyManifest, err
	}

	for _, m := range manifests {
		if m.Name == manifestName {
			return m, nil
		}
	}

	return emptyManifest, fmt.Errorf("did not find any manifest matching name '%s'", manifestName)
}

func unmarshalAllManifests(in []byte, out *[]Manifest) error {
	r := bytes.NewReader(in)
	decoder := yaml.NewDecoder(r)
	for {
		var m Manifest
		if err := decoder.Decode(&m); err != nil {
			// Break when there are no more documents to decode
			if err != io.EOF {
				return err
			}
			break
		}
		*out = append(*out, m)
	}
	return nil
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

func checkDuplicateManifests(manifests []Manifest) error {
	keys := make(map[string]bool)
	for _, entry := range manifests {
		if _, value := keys[entry.Name]; value {
			return fmt.Errorf("found manifests with duplicate names: %s", entry.Name)
		} else {
			keys[entry.Name] = true
		}
	}
	return nil
}
