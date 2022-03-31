package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_NoConfig(t *testing.T) {
	u, _ := uuid.NewUUID()
	testConfigPath := fmt.Sprintf("/tmp/.pi-app-deployer.app.%s.yaml", u.String())
	appConfigs, err := GetAppConfigs(testConfigPath)
	assert.NoError(t, err)
	assert.NotNil(t, appConfigs)
	assert.NotNil(t, appConfigs.Map)
	assert.Equal(t, map[string]Config{}, appConfigs.Map)
}

func Test_CreateConfig(t *testing.T) {
	c := Config{
		RepoName:      "andrewmarklloyd/pi-test",
		ManifestName:  "pi-test-arm",
		AppUser:       "pi",
		LogForwarding: false,
		EnvVars:       map[string]string{"MY_CONFIG": "foobar", "HELLO_CONFIG": "testing"},
	}
	a := AppConfigs{Map: map[string]Config{"andrewmarklloyd_pi-test_pi-test-arm": c}}

	u, _ := uuid.NewUUID()
	testConfigPath := fmt.Sprintf("/tmp/.pi-app-deployer.app.%s.yaml", u.String())

	err := a.WriteAppConfigs(testConfigPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(testConfigPath)
	assert.NoError(t, err)
	expectedContent := `map:
  andrewmarklloyd_pi-test_pi-test-arm:
    repoName: andrewmarklloyd/pi-test
    manifestName: pi-test-arm
    appUser: pi
    logForwarding: false
    envVars:
      HELLO_CONFIG: testing
      MY_CONFIG: foobar
`
	assert.Equal(t, expectedContent, string(content))

	aConf, err := GetAppConfigs(testConfigPath)
	assert.NoError(t, err)

	actual := aConf.Map["andrewmarklloyd_pi-test_pi-test-arm"]

	assert.Equal(t, "pi", actual.AppUser)
	assert.Equal(t, "andrewmarklloyd/pi-test", actual.RepoName)
	assert.Equal(t, "pi-test-arm", actual.ManifestName)
	assert.False(t, actual.LogForwarding)
	expectedMap := make(map[string]string)
	expectedMap["MY_CONFIG"] = "foobar"
	expectedMap["HELLO_CONFIG"] = "testing"
	assert.Equal(t, expectedMap, actual.EnvVars)
}

func Test_CreateMultipleConfigs(t *testing.T) {
	c1 := Config{
		RepoName:      "andrewmarklloyd/pi-test",
		ManifestName:  "pi-test-arm",
		AppUser:       "pi",
		LogForwarding: false,
		EnvVars:       map[string]string{"MY_CONFIG": "foobar", "HELLO_CONFIG": "testing"},
	}

	c2 := Config{
		RepoName:      "andrewmarklloyd/pi-test-2",
		ManifestName:  "pi-test-amd64",
		AppUser:       "app-runner",
		LogForwarding: true,
		EnvVars:       map[string]string{"HELLO_WORLD": "hello-world", "CONFIG": "config-test"},
	}

	appConfigs := AppConfigs{
		map[string]Config{
			configToKey(c1): c1,
			configToKey(c2): c2,
		},
	}

	u, _ := uuid.NewUUID()
	testConfigPath := fmt.Sprintf("/tmp/.pi-app-deployer.app.%s.yaml", u.String())
	err := appConfigs.WriteAppConfigs(testConfigPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(testConfigPath)
	assert.NoError(t, err)
	expectedContent := `map:
  andrewmarklloyd_pi-test-2_pi-test-amd64:
    repoName: andrewmarklloyd/pi-test-2
    manifestName: pi-test-amd64
    appUser: app-runner
    logForwarding: true
    envVars:
      CONFIG: config-test
      HELLO_WORLD: hello-world
  andrewmarklloyd_pi-test_pi-test-arm:
    repoName: andrewmarklloyd/pi-test
    manifestName: pi-test-arm
    appUser: pi
    logForwarding: false
    envVars:
      HELLO_CONFIG: testing
      MY_CONFIG: foobar
`
	assert.Equal(t, expectedContent, string(content))

	appConfigs, err = GetAppConfigs(testConfigPath)
	assert.NoError(t, err)

	c1Actual := appConfigs.Map["andrewmarklloyd_pi-test_pi-test-arm"]
	assert.Equal(t, "pi", c1Actual.AppUser)
	assert.Equal(t, "andrewmarklloyd/pi-test", c1Actual.RepoName)
	assert.Equal(t, "pi-test-arm", c1Actual.ManifestName)
	assert.False(t, c1Actual.LogForwarding)
	expectedMap := make(map[string]string)
	expectedMap["MY_CONFIG"] = "foobar"
	expectedMap["HELLO_CONFIG"] = "testing"
	assert.Equal(t, expectedMap, c1Actual.EnvVars)

	c2Actual := appConfigs.Map["andrewmarklloyd_pi-test-2_pi-test-amd64"]
	assert.Equal(t, "app-runner", c2Actual.AppUser)
	assert.Equal(t, "andrewmarklloyd/pi-test-2", c2Actual.RepoName)
	assert.Equal(t, "pi-test-amd64", c2Actual.ManifestName)
	assert.True(t, c2Actual.LogForwarding)
	expectedMap = make(map[string]string)
	expectedMap["CONFIG"] = "config-test"
	expectedMap["HELLO_WORLD"] = "hello-world"
	assert.Equal(t, expectedMap, c2Actual.EnvVars)
}

func Test_ConfigExists(t *testing.T) {
	c1 := Config{
		RepoName:      "andrewmarklloyd/pi-test",
		ManifestName:  "pi-test-arm",
		AppUser:       "pi",
		LogForwarding: false,
		EnvVars:       map[string]string{"MY_CONFIG": "foobar", "HELLO_CONFIG": "testing"},
	}
	c2 := Config{
		RepoName:      "andrewmarklloyd/pi-test-2",
		ManifestName:  "pi-test-amd64",
		AppUser:       "app-runner",
		LogForwarding: true,
		EnvVars:       map[string]string{"HELLO_WORLD": "hello-world", "CONFIG": "config-test"},
	}
	c3 := Config{
		RepoName:      "andrewmarklloyd/pi-test",
		ManifestName:  "pi-agent-arm",
		AppUser:       "app-runner",
		LogForwarding: true,
		EnvVars:       map[string]string{"HELLO_WORLD": "hello-world", "CONFIG": "config-test"},
	}

	appConfigs := AppConfigs{
		map[string]Config{
			configToKey(c1): c1,
			configToKey(c2): c2,
		},
	}
	exists := appConfigs.ConfigExists(c1)
	assert.True(t, exists, "Config should exist in the app configs struct")

	exists = appConfigs.ConfigExists(c3)
	assert.False(t, exists, "Config should NOT exist in the app configs struct")
}

func Test_configToKey(t *testing.T) {
	c1 := Config{
		RepoName:      "andrewmarklloyd/pi-test",
		ManifestName:  "pi-test-arm",
		AppUser:       "pi",
		LogForwarding: false,
		EnvVars:       map[string]string{"MY_CONFIG": "foobar", "HELLO_CONFIG": "testing"},
	}
	k := configToKey(c1)
	assert.Equal(t, "andrewmarklloyd_pi-test_pi-test-arm", k)
}
