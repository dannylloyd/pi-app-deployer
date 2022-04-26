package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Keys(t *testing.T) {
	key := getAgentInventoryWriteKey("my-repo", "my-manifest", "host-1")
	assert.Equal(t, "agent/inventory/my-repo/my-manifest/host-1", key)

	key = getWriteKey("my-repo", "my-manifest", "host-1")
	assert.Equal(t, "repo/push/status/my-repo/my-manifest/host-1", key)

	key = getReadKey("my-repo", "my-manifest")
	assert.Equal(t, "repo/push/status/my-repo/my-manifest/*", key)

	key = getAgentInventoryReadKey("my-repo", "my-manifest")
	assert.Equal(t, "agent/inventory/my-repo/my-manifest/*", key)
}
