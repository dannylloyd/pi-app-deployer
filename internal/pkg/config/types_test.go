package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ValidateArtifact(t *testing.T) {
	sha := "35341f353d050061e4af496bbcc95f4d6fd7ea79"

	validArtifact := Artifact{
		SHA:          sha,
		Repository:   "andrewmarklloyd/pi-test",
		Name:         fmt.Sprintf("app_%s", sha),
		ManifestName: "pi-test",
	}

	err := validArtifact.Validate()
	assert.NoError(t, err)

	invalidArtifact := Artifact{}

	err = invalidArtifact.Validate()
	expectedErr := `4 errors occurred:
	* repository field is required
	* name field is required
	* sha field is required
	* manifest_name field is required

`
	assert.Equal(t, err.Error(), expectedErr)
}
