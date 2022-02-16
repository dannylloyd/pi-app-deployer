package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Version(t *testing.T) {
	packageName := "pi-test"
	vTool := NewVersionTool(true, packageName)

	err := vTool.Cleanup()
	assert.NoError(t, err)

	installed, err := vTool.AppInstalled()
	assert.NoError(t, err)
	assert.False(t, installed)

	_, err = vTool.GetCurrentVersion()
	assert.EqualError(t, err, "reading current version from file: open /tmp/.pi-test.version: no such file or directory")

	err = vTool.WriteCurrentVersion("v1.0.5")
	assert.NoError(t, err)

	version, err := vTool.GetCurrentVersion()
	assert.NoError(t, err)
	assert.Equal(t, version, "v1.0.5")

	err = vTool.Cleanup()
	assert.NoError(t, err)
}
