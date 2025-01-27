// Copyright 2020 PingCAP, Inc.
// Modifications copyright (C) 2021 MatrixOrigin.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package statistics

import (
	"github.com/matrixorigin/matrixcube/components/prophet/core"
)

// ResourceStats records a list of resources' statistics and distribution status.
type ResourceStats struct {
	Count                int              `json:"count"`
	EmptyCount           int              `json:"empty_count"`
	StorageSize          int64            `json:"storage_size"`
	StorageKeys          int64            `json:"storage_keys"`
	ContainerLeaderCount map[uint64]int   `json:"container_leader_count"`
	ContainerPeerCount   map[uint64]int   `json:"container_peer_count"`
	ContainerLeaderSize  map[uint64]int64 `json:"container_leader_size"`
	ContainerLeaderKeys  map[uint64]int64 `json:"container_leader_keys"`
	ContainerPeerSize    map[uint64]int64 `json:"container_peer_size"`
	ContainerPeerKeys    map[uint64]int64 `json:"container_peer_keys"`
}

// GetResourceStats sums resources' statistics.
func GetResourceStats(resources []*core.CachedResource) *ResourceStats {
	stats := newResourceStats()
	for _, resource := range resources {
		stats.Observe(resource)
	}
	return stats
}

func newResourceStats() *ResourceStats {
	return &ResourceStats{
		ContainerLeaderCount: make(map[uint64]int),
		ContainerPeerCount:   make(map[uint64]int),
		ContainerLeaderSize:  make(map[uint64]int64),
		ContainerLeaderKeys:  make(map[uint64]int64),
		ContainerPeerSize:    make(map[uint64]int64),
		ContainerPeerKeys:    make(map[uint64]int64),
	}
}

// Observe adds a resource's statistics into ResourceStats.
func (s *ResourceStats) Observe(r *core.CachedResource) {
	s.Count++
	approximateKeys := r.GetApproximateKeys()
	approximateSize := r.GetApproximateSize()
	if approximateSize <= core.EmptyResourceApproximateSize {
		s.EmptyCount++
	}
	s.StorageSize += approximateSize
	s.StorageKeys += approximateKeys
	leader := r.GetLeader()
	if leader != nil {
		containerID := leader.GetContainerID()
		s.ContainerLeaderCount[containerID]++
		s.ContainerLeaderSize[containerID] += approximateSize
		s.ContainerLeaderKeys[containerID] += approximateKeys
	}
	peers := r.Meta.Peers()
	for _, p := range peers {
		containerID := p.GetContainerID()
		s.ContainerPeerCount[containerID]++
		s.ContainerPeerSize[containerID] += approximateSize
		s.ContainerPeerKeys[containerID] += approximateKeys
	}
}
