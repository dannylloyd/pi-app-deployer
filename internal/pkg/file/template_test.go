package file

import (
	"testing"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_ServiceTemplate(t *testing.T) {

	m, err := manifest.GetManifest("../../../test/templates/fully-defined-manifest.yaml")
	assert.NoError(t, err)

	serviceFile, err := EvalServiceTemplate(m)
	assert.NoError(t, err)

	expectedServiceFile := `[Unit]
Description=Sample App
After=a.service b.service
Requires=c.service
StartLimitInterval=300
StartLimitBurst=10

[Install]
WantedBy=multi-user.target
StandardOutput=inherit
[Service]
ExecStart=/home/pi/run-sample-app.sh
WorkingDirectory=/home/pi/
StandardOutput=inherit
StandardError=inherit
TimeoutStartSec=7
Restart=always
RestartSec=23
User=pi
Environment=HEROKU_API_KEY={{.HerokuAPIKey}}
`
	assert.Equal(t, expectedServiceFile, serviceFile)
}

func Test_EvalUpdaterTemplate(t *testing.T) {
	c := config.Config{
		RepoName:    "andrewmarklloyd/pi-test",
		PackageName: "pi-test-client",
	}
	serviceFile, err := EvalUpdaterTemplate(c)
	assert.NoError(t, err)

	expectedServiceFile := `[Unit]
Description=pi-app-updater
After=network.target
StartLimitInterval=300
StartLimitBurst=10

[Install]
WantedBy=multi-user.target

[Service]
ExecStart=/home/pi/pi-app-updater --repo-name andrewmarklloyd/pi-test --package-name pi-test-client
WorkingDirectory=/home/pi/
StandardOutput=inherit
StandardError=inherit
Restart=always
RestartSec=30
User=root
`

	assert.Equal(t, expectedServiceFile, serviceFile)
}

func Test_EvalUpdaterTemplateErrs(t *testing.T) {
	c := config.Config{}
	serviceFile, err := EvalUpdaterTemplate(c)
	assert.Empty(t, serviceFile)
	expectedErr := `2 errors occurred:
	* config package name is required
	* config repo name is required

`
	assert.Equal(t, err.Error(), expectedErr)
}

func Test_EvalRunScriptTemplate(t *testing.T) {
	// TODO need to have heroku client as interface to implement mock API call
}
