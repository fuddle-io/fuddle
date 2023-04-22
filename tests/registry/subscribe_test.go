//go:build all || integration

package registry

import (
	"context"
	"fmt"
	"testing"
	"time"

	rpc "github.com/fuddle-io/fuddle-rpc/go"
	"github.com/fuddle-io/fuddle/pkg/fcm/cluster"
	"github.com/fuddle-io/fuddle/pkg/registry"
	registryClient "github.com/fuddle-io/fuddle/pkg/registry/client"
	"github.com/fuddle-io/fuddle/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscribe_ReceiveNodesLocalMember(t *testing.T) {
	c, err := cluster.NewCluster(cluster.WithFuddleNodes(1))
	require.Nil(t, err)
	defer c.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	assert.NoError(t, c.WaitForHealthy(ctx))

	node := c.FuddleNodes()[0]

	r := registry.NewRegistry("local")

	updatesCh := make(chan *rpc.Member2)
	req := &rpc.SubscribeRequest{}
	r.Subscribe(req, func(update *rpc.Member2) {
		updatesCh <- update
	})

	client, err := registryClient.Connect(
		node.Fuddle.Config.RPC.JoinAdvAddr(),
		r,
		registryClient.WithLogger(testutils.Logger()),
	)
	require.NoError(t, err)
	defer client.Close()

	// Wait to receive the servers local node.
	u, err := waitForUpdate(updatesCh)
	assert.NoError(t, err)
	assert.Equal(t, node.Fuddle.Config.NodeID, u.State.Id)
}

func waitForUpdate(ch <-chan *rpc.Member2) (*rpc.Member2, error) {
	select {
	case u := <-ch:
		return u, nil
	case <-time.After(time.Second):
		return nil, fmt.Errorf("timeout")
	}
}
