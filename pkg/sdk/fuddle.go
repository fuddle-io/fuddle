// Copyright (C) 2023 Andrew Dunstall
//
// Fuddle is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Fuddle is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package sdk

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/andydunstall/fuddle/pkg/registry"
	"github.com/andydunstall/fuddle/pkg/rpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Attributes struct {
	ID       string
	Service  string
	Locality string
	Revision string
}

type Fuddle struct {
	id string

	nodeMap *registry.NodeMap

	conn   *grpc.ClientConn
	stream rpc.Registry_RegisterClient

	wg sync.WaitGroup

	logger *zap.Logger
}

func Register(addr string, attrs Attributes, state map[string]string, logger *zap.Logger) (*Fuddle, error) {
	conn, err := grpc.Dial(
		addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("fuddle: %w", err)
	}

	client := rpc.NewRegistryClient(conn)
	stream, err := client.Register(context.Background())
	if err != nil {
		return nil, fmt.Errorf("fuddle: %w", err)
	}

	joinUpdate := &rpc.NodeUpdate{
		NodeId:     attrs.ID,
		UpdateType: rpc.UpdateType_NODE_JOIN,
		Attributes: &rpc.Attributes{
			Id:       attrs.ID,
			Service:  attrs.Service,
			Locality: attrs.Locality,
			Revision: attrs.Revision,
		},
		State: state,
	}

	nodeMap := registry.NewNodeMap(registry.NodeState{
		ID:       attrs.ID,
		Service:  attrs.Service,
		Locality: attrs.Locality,
		Revision: attrs.Revision,
		State:    state,
	})
	if err := nodeMap.Update(joinUpdate); err != nil {
		conn.Close()
		return nil, fmt.Errorf("fuddle: %w", err)
	}
	if err := stream.Send(joinUpdate); err != nil {
		conn.Close()
		return nil, fmt.Errorf("fuddle: %w", err)
	}

	f := &Fuddle{
		id:      attrs.ID,
		nodeMap: nodeMap,
		conn:    conn,
		stream:  stream,
		logger:  logger,
	}

	f.wg.Add(1)
	go f.sync()

	return f, nil
}

func (f *Fuddle) ID() string {
	return f.id
}

func (f *Fuddle) Nodes() []registry.NodeState {
	return f.nodeMap.Nodes()
}

func (f *Fuddle) Update(key string, value string) error {
	update := &rpc.NodeUpdate{
		NodeId:     f.id,
		UpdateType: rpc.UpdateType_NODE_UPDATE,
		State: map[string]string{
			key: value,
		},
	}
	if err := f.nodeMap.Update(update); err != nil {
		return fmt.Errorf("fuddle: %w", err)
	}
	if err := f.stream.Send(update); err != nil {
		return fmt.Errorf("fuddle: %w", err)
	}
	return nil
}

func (f *Fuddle) Subscribe(rewind bool, cb func(update *rpc.NodeUpdate)) func() {
	return f.nodeMap.Subscribe(rewind, cb)
}

func (f *Fuddle) Unregister() error {
	err := f.stream.Send(&rpc.NodeUpdate{
		NodeId:     f.id,
		UpdateType: rpc.UpdateType_NODE_LEAVE,
	})

	f.conn.Close()
	f.wg.Wait()
	return err
}

func (f *Fuddle) sync() {
	defer f.wg.Done()

	for {
		update, err := f.stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			return
		}

		if err := f.nodeMap.Update(update); err != nil {
			f.logger.Error("failed to update state", zap.Error(err))
		}
	}
}
