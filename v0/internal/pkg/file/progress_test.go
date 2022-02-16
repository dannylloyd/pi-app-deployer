package file

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var wg sync.WaitGroup

func Test_Progress(t *testing.T) {
	err := SetUpdateInProgress(true)
	assert.NoError(t, nil, err)
	inProgress := UpdateInProgress()
	assert.True(t, inProgress)

	err = SetUpdateInProgress(false)
	assert.NoError(t, nil, err)

	inProgress = UpdateInProgress()
	assert.False(t, inProgress)
}
