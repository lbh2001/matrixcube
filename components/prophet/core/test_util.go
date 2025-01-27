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

package core

import (
	"math"

	"github.com/matrixorigin/matrixcube/components/prophet/metadata"
	"github.com/matrixorigin/matrixcube/components/prophet/pb/metapb"
)

var (
	testResourceFactory = func() metadata.Resource {
		return &metadata.TestResource{}
	}
)

// SplitTestResources split a set of CachedResource by the middle of resourceKey
func SplitTestResources(resources []*CachedResource) []*CachedResource {
	results := make([]*CachedResource, 0, len(resources)*2)
	for _, res := range resources {
		resStart, resEnd := res.Meta.Range()
		start, end := byte(0), byte(math.MaxUint8)
		if len(resStart) > 0 {
			start = resStart[0]
		}
		if len(resEnd) > 0 {
			end = resEnd[0]
		}
		middle := []byte{start/2 + end/2}

		left := res.Clone()
		left.Meta.SetID(res.Meta.ID() + uint64(len(resources)))
		left.Meta.SetEndKey(middle)
		epoch := left.Meta.Epoch()
		epoch.Version++
		left.Meta.SetEpoch(epoch)

		right := res.Clone()
		right.Meta.SetID(res.Meta.ID() + uint64(len(resources)*2))
		right.Meta.SetStartKey(middle)
		epoch = right.Meta.Epoch()
		epoch.Version++
		right.Meta.SetEpoch(epoch)
		results = append(results, left, right)
	}
	return results
}

// MergeTestResources merge a set of CachedResource by resourceKey
func MergeTestResources(resources []*CachedResource) []*CachedResource {
	results := make([]*CachedResource, 0, len(resources)/2)
	for i := 0; i < len(resources); i += 2 {
		left := resources[i]
		right := resources[i]
		if i+1 < len(resources) {
			right = resources[i+1]
		}

		leftStart, _ := left.Meta.Range()
		_, rightEnd := right.Meta.Range()
		res := &CachedResource{Meta: &metadata.TestResource{
			ResID: left.Meta.ID() + uint64(len(resources)),
			Start: leftStart,
			End:   rightEnd,
		}}
		if left.Meta.Epoch().Version > right.Meta.Epoch().Version {
			res.Meta.SetEpoch(left.Meta.Epoch())
		} else {
			res.Meta.SetEpoch(right.Meta.Epoch())
		}

		epoch := res.Meta.Epoch()
		epoch.Version++
		res.Meta.SetEpoch(epoch)
		results = append(results, res)
	}
	return results
}

// NewTestCachedResource creates a CachedResource for test.
func NewTestCachedResource(start, end []byte) *CachedResource {
	return &CachedResource{Meta: &metadata.TestResource{
		Start:    start,
		End:      end,
		ResEpoch: metapb.ResourceEpoch{},
	}}
}

// NewTestContainerInfoWithLabel is create a container with specified labels.
func NewTestContainerInfoWithLabel(id uint64, resourceCount int, labels map[string]string) *CachedContainer {
	containerLabels := make([]metapb.Pair, 0, len(labels))
	for k, v := range labels {
		containerLabels = append(containerLabels, metapb.Pair{
			Key:   k,
			Value: v,
		})
	}
	stats := &metapb.ContainerStats{}
	stats.Capacity = uint64(1024)
	stats.Available = uint64(1024)
	container := NewCachedContainer(
		&metadata.TestContainer{
			CID:     id,
			CLabels: containerLabels,
		},
		SetContainerStats(stats),
		SetResourceCount(0, resourceCount),
		SetResourceSize(0, int64(resourceCount)*10),
	)
	return container
}

// NewTestCachedContainerWithSizeCount is create a container with size and count.
func NewTestCachedContainerWithSizeCount(id uint64, resourceCount, leaderCount int, resourceSize, leaderSize int64) *CachedContainer {
	stats := &metapb.ContainerStats{}
	stats.Capacity = uint64(1024)
	stats.Available = uint64(1024)
	container := NewCachedContainer(
		&metadata.TestContainer{
			CID: id,
		},
		SetContainerStats(stats),
		SetResourceCount(0, resourceCount),
		SetResourceSize(0, resourceSize),
		SetLeaderCount(0, leaderCount),
		SetLeaderSize(0, leaderSize),
	)
	return container
}

func newTestResourceItem(start, end []byte) *resourceItem {
	return &resourceItem{res: NewTestCachedResource(start, end)}
}

func newResourceWithStat(start, end string, size, keys int64) *CachedResource {
	res := NewTestCachedResource([]byte(start), []byte(end))
	res.stats.ApproximateSize, res.stats.ApproximateKeys = uint64(size), uint64(keys)
	return res
}
