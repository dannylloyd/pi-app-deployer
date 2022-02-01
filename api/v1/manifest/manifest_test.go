package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FullyDefinedManifest(t *testing.T) {
	m, err := GetManifest("../../../test/templates/fully-defined-manifest.yaml")
	if err != nil {
		t.Error("getting fully defined manifest should not err, got ", err)
	}

	assert.Equal(t, "sample-app", m.Name)
	assert.Equal(t, "sample-app-test", m.Heroku.App)
	assert.Equal(t, []string{"CLOUDMQTT_URL", "LOG_LEVEL"}, m.Heroku.Env)
	assert.Equal(t, "Sample App", m.Systemd.Unit.Description)
	assert.Equal(t, []string{"a.service", "b.service"}, m.Systemd.Unit.After)
	assert.Equal(t, []string{"c.service"}, m.Systemd.Unit.Requires)
	assert.Equal(t, 7, m.Systemd.Service.TimeoutStartSec)
	assert.Equal(t, "always", m.Systemd.Service.Restart)
	assert.Equal(t, 23, m.Systemd.Service.RestartSec)
}

func Test_FullyDefaults(t *testing.T) {
	m, err := GetManifest("../../../test/templates/minimally-defined-manifest.yaml")
	if err != nil {
		t.Error("getting fully defined manifest should not err, got ", err)
	}

	assert.Equal(t, "sample-app", m.Name)
	assert.Equal(t, "sample-app-test", m.Heroku.App)

	assert.Equal(t, "sample-app", m.Systemd.Unit.Description)
	assert.Equal(t, []string{"systemd-journald.service", "network.target"}, m.Systemd.Unit.After)
	assert.Equal(t, []string{"systemd-journald.service"}, m.Systemd.Unit.Requires)
	assert.Equal(t, 0, m.Systemd.Service.TimeoutStartSec)
	assert.Equal(t, "on-failure", m.Systemd.Service.Restart)
	assert.Equal(t, 5, m.Systemd.Service.RestartSec)
}
