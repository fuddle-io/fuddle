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

package registry

// NodeState represents the state of a node that is propagated to other nodes
// in the cluster.
type NodeState struct {
	// ID is a unique identifier for the node in the cluster.
	ID string `json:"id,omitempty"`

	// Service is the type of service running on the node.
	Service string `json:"service,omitempty"`

	// Locality is the location of the node in the cluster.
	Locality string `json:"locality,omitempty"`

	// Created is the time the node was created in UNIX milliseconds.
	Created int64

	// Revision identifies the version of the service running on the node.
	Revision string `json:"revision,omitempty"`

	// State node service state.
	State map[string]string `json:"state,omitempty"`
}

func (s *NodeState) Copy() NodeState {
	cp := *s
	cp.State = CopyState(s.State)
	return cp
}

func CopyState(s map[string]string) map[string]string {
	if s == nil {
		return make(map[string]string)
	}

	cp := make(map[string]string)
	for k, v := range s {
		cp[k] = v
	}
	return cp
}
