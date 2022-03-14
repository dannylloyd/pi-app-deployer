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
	expectedErr := `4 errors occurred:\n\t* repository field is required\n\t* name field is required\n\t* sha field is required\n\t* manifest_name field is required\n\n`
	assert.Equal(t, err.Error(), expectedErr)
}

func Test_ValidateServicActionPayload(t *testing.T) {

	validPayload := ServiceActionPayload{
		RepoName:     "andrewmarklloyd/test",
		ManifestName: "test",
		Action:       "RESTART",
	}

	err := validPayload.Validate()
	assert.NoError(t, err)

	invalidPayload := ServiceActionPayload{}

	err = invalidPayload.Validate()
	assert.Error(t, err)
	expectedErr := `3 errors occurred:\n\t* repoName field is required\n\t* manifestName field is required\n\t* action field is required\n\n`
	assert.Equal(t, err.Error(), expectedErr)

	invalidPayload = ServiceActionPayload{
		Action: "restart",
	}

	err = invalidPayload.Validate()
	assert.Error(t, err)
	expectedErr = `3 errors occurred:\n\t* repoName field is required\n\t* manifestName field is required\n\t* action must be one of: START, STOP, or RESTART, but was restart\n\n`
	assert.Equal(t, err.Error(), expectedErr)
}
