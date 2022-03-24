package file

import (
	"os"
	"testing"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_ServiceTemplate(t *testing.T) {

	m, err := manifest.GetManifest("../../../test/templates/fully-defined-manifest.yaml", "sample-app")
	assert.NoError(t, err)

	serviceFile, err := EvalServiceTemplate(m, "/home/pi", "pi")
	assert.NoError(t, err)

	expectedServiceFile := `[Unit]
Description=Sample App
After=a.service b.service
Requires=c.service
StartLimitInterval=300
StartLimitBurst=10

[Install]
WantedBy=multi-user.target

[Service]
EnvironmentFile=/home/pi/.sample-app.env
ExecStart=/home/pi/run-sample-app.sh
WorkingDirectory=/home/pi
StandardOutput=inherit
StandardError=inherit
TimeoutStartSec=7
Restart=always
RestartSec=23
User=pi
`
	assert.Equal(t, expectedServiceFile, serviceFile)
}

func Test_EvalDeployerTemplate(t *testing.T) {
	c := config.Config{
		RepoName:     "andrewmarklloyd/pi-test",
		ManifestName: "pi-test",
		HomeDir:      "/home/pi",
		AppUser:      "pi",
	}
	serviceFile, err := EvalDeployerTemplate(c)
	assert.NoError(t, err)

	expectedServiceFile := `[Unit]
Description=pi-app-deployer-agent
After=network.target
StartLimitInterval=300
StartLimitBurst=10

[Install]
WantedBy=multi-user.target

[Service]
EnvironmentFile=/home/pi/.pi-app-deployer-agent.env
ExecStart=/home/pi/pi-app-deployer-agent --repo-name andrewmarklloyd/pi-test --manifest-name pi-test
WorkingDirectory=/home/pi
StandardOutput=inherit
StandardError=inherit
Restart=always
RestartSec=30
User=root
`

	assert.Equal(t, expectedServiceFile, serviceFile)
}

func Test_EvalDeployerTemplateErrs(t *testing.T) {
	c := config.Config{}
	serviceFile, err := EvalDeployerTemplate(c)
	assert.Empty(t, serviceFile)
	expectedErr := `2 errors occurred:
	* config repo name is required
	* config manifest name is required

`
	assert.Equal(t, err.Error(), expectedErr)
}

func Test_EvalRunScriptTemplate(t *testing.T) {
	m, err := manifest.GetManifest("../../../test/templates/fully-defined-manifest.yaml", "sample-app")
	assert.NoError(t, err)

	runScriptFile, err := EvalRunScriptTemplate(m, "b1946ac92492d2347c6235b4d2611184", "/home/pi")
	assert.NoError(t, err)

	expectedRunScriptFile := `#!/bin/bash

export APP_VERSION=b1946ac92492d2347c6235b4d2611184


if [[ -z ${HEROKU_API_KEY} ]]; then
  echo "HEROKU_API_KEY env var not set, exiting now"
  exit 1
fi

vars=$(curl -s -n https://api.heroku.com/apps/sample-app-test/config-vars \
  -H "Accept: application/vnd.heroku+json; version=3" \
  -H "Authorization: Bearer ${HEROKU_API_KEY}")

export CLOUDMQTT_URL=$(echo $vars | jq -r '.CLOUDMQTT_URL')
if [[ -z ${CLOUDMQTT_URL} || ${CLOUDMQTT_URL} == 'null' ]]; then
  echo "CLOUDMQTT_URL env var not set, exiting now"
  exit 1
fi

export LOG_LEVEL=$(echo $vars | jq -r '.LOG_LEVEL')
if [[ -z ${LOG_LEVEL} || ${LOG_LEVEL} == 'null' ]]; then
  echo "LOG_LEVEL env var not set, exiting now"
  exit 1
fi


unset HEROKU_API_KEY

/home/pi/sample-app-agent
`

	assert.Equal(t, expectedRunScriptFile, runScriptFile)
}

func Test_Helpers(t *testing.T) {
	c := config.Config{
		RepoName:     "andrewmarklloyd/pi-test",
		ManifestName: "pi-test",
		HomeDir:      "/home/pi",
		AppUser:      "pi",
	}
	expected := "/home/pi/pi-app-deployer-agent --repo-name andrewmarklloyd/pi-test --manifest-name pi-test"
	actual := getDeployerExecStart(c)
	assert.Equal(t, expected, actual)

	c = config.Config{
		RepoName:      "andrewmarklloyd/pi-test",
		ManifestName:  "pi-test",
		HomeDir:       "/home/pi",
		AppUser:       "pi",
		LogForwarding: true,
	}
	expected = "/home/pi/pi-app-deployer-agent --repo-name andrewmarklloyd/pi-test --manifest-name pi-test --log-forwarding"
	actual = getDeployerExecStart(c)
	assert.Equal(t, expected, actual)
}

func Test_WriteServiceEnvFile(t *testing.T) {
	m, err := manifest.GetManifest("../../../test/templates/fully-defined-manifest.yaml", "sample-app")
	assert.NoError(t, err)

	envVars := map[string]string{
		"MY_CONFIG":    "testing",
		"EXTRA_CONFIG": "foobar",
	}

	cfg := config.Config{
		HomeDir: "/tmp",
		EnvVars: envVars,
	}
	err = WriteServiceEnvFile(m, "abcdefg", "hijklmn", cfg)
	assert.NoError(t, err)
	b, err := os.ReadFile("/tmp/.sample-app.env")
	assert.NoError(t, err)
	assert.Equal(t, "HEROKU_API_KEY=abcdefg\nAPP_VERSION=hijklmn\nEXTRA_CONFIG=foobar\nMY_CONFIG=testing", string(b))
}
