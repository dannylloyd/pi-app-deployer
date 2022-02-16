package file

import (
	"fmt"
	"os"
	"testing"

	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
	"github.com/stretchr/testify/assert"
)

func Test_Systemd(t *testing.T) {
	testMode := true
	cfg := config.Config{
		RepoName:    "andrewmarklloyd/pi-test",
		PackageName: "pi-test-client",
	}
	sdTool := NewSystemdTool(testMode, cfg)
	assert.Equal(t, "pi-test-client.service", sdTool.UnitName)
	assert.Equal(t, "/tmp/pi-test-client/pi-test-client.service", sdTool.UnitPath)

	err := os.RemoveAll("/tmp/pi-test-client")
	assert.NoError(t, err)
	err = os.Mkdir("/tmp/pi-test-client", 0755)
	assert.NoError(t, err)

	serviceFileString := `[Unit]
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
Environment=HEROKU_API_KEY=abcdefg
`

	err = os.WriteFile(sdTool.UnitPath, []byte(serviceFileString), 0644)
	fmt.Println(err)

	apiKey, err := sdTool.FindApiKeyFromSystemd()
	assert.NoError(t, err)
	assert.Equal(t, "abcdefg", apiKey)

	err = os.RemoveAll("/tmp/pi-test-client")
	assert.NoError(t, err)
}
