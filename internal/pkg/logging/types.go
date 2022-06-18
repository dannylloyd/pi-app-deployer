package logging

type ConfigMap map[string]LogForwardConfig

type LogForwardConfig struct {
	ApiKey   string `yaml:"apiKey"`
	Endpoint string `yaml:"endpoint"`
}
