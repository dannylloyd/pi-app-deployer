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

	serviceFile, err := EvalServiceTemplate(m, "pi")
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
EnvironmentFile=/usr/local/src/pi-app-deployer/.sample-app.env
ExecStart=/usr/local/src/pi-app-deployer/run-sample-app.sh
WorkingDirectory=/usr/local/src/pi-app-deployer
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
	serviceFile, err := EvalDeployerTemplate("heroku-app")
	assert.NoError(t, err)

	expectedServiceFile := `[Unit]
Description=pi-app-deployer-agent
After=network.target
StartLimitInterval=300
StartLimitBurst=10

[Install]
WantedBy=multi-user.target

[Service]
EnvironmentFile=/usr/local/src/pi-app-deployer/.pi-app-deployer-agent.env
ExecStart=/usr/local/src/pi-app-deployer/pi-app-deployer-agent update --herokuApp heroku-app
WorkingDirectory=/usr/local/src/pi-app-deployer
StandardOutput=inherit
StandardError=inherit
Restart=always
RestartSec=30
User=root
`

	assert.Equal(t, expectedServiceFile, serviceFile)
}

func Test_EvalRunScriptTemplate(t *testing.T) {
	m, err := manifest.GetManifest("../../../test/templates/fully-defined-manifest.yaml", "sample-app")
	assert.NoError(t, err)

	runScriptFile, err := EvalRunScriptTemplate(m, "b1946ac92492d2347c6235b4d2611184")
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

/usr/local/src/pi-app-deployer/sample-app-agent
`

	assert.Equal(t, expectedRunScriptFile, runScriptFile)
}

func Test_Helpers(t *testing.T) {
	expected := "/usr/local/src/pi-app-deployer/pi-app-deployer-agent update --herokuApp testing-app"
	actual := getDeployerExecStart("testing-app")
	assert.Equal(t, expected, actual)

	expected = "/usr/local/src/pi-app-deployer/pi-app-deployer-agent update --herokuApp testing-app"
	actual = getDeployerExecStart("testing-app")
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
		EnvVars: envVars,
	}
	err = WriteServiceEnvFile(m, "abcdefg", "hijklmn", cfg, "/tmp")
	assert.NoError(t, err)
	b, err := os.ReadFile("/tmp/.sample-app.env")
	assert.NoError(t, err)
	assert.Equal(t, "HEROKU_API_KEY=abcdefg\nAPP_VERSION=hijklmn\nEXTRA_CONFIG=foobar\nMY_CONFIG=testing", string(b))
}
