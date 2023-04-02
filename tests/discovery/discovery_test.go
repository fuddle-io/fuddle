//go:build all || integration

package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/fuddle-io/fuddle/tests/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests creating a 5 node cluster and waiting for each node to discover one
// another.
func TestDiscovery_Discovery(t *testing.T) {
	c, err := cluster.NewCluster(cluster.WithNodes(5))
	require.Nil(t, err)
	defer c.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	assert.NoError(t, c.WaitForHealthy(ctx))
}
