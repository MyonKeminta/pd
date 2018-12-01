package server

import (
	"bytes"
	"sort"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server/core"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	timestamp int64 // unix-nano
	leader    uint64
	eventType string

	meta *metapb.Region

	// make sure the range of pointing regions are in ascend order
	// int is index of nodes
	parents []int
	// cause it is no need to send to front end, no need to be in order
	// int is index of nodes
	children []int
}

type regionHistory struct {
	nodes []*Node
	kv    *core.KV

	// region_id -> index of nodes
	latest map[uint64]int
}

func newRegionHistory(kv *core.KV) *regionHistory {
	return &regionHistory{
		kv:     kv,
		nodes:  make([]*Node, 0),
		latest: make(map[uint64]int),
	}
}

func (h *regionHistory) onRegionSplit(originID uint64, regions []*metapb.Region) {
	index, ok := h.latest[originID]
	if !ok {
		log.Errorf("[Split] not found latest info of region %v", originID)
		return
	}
	origin := h.nodes[index]

	now := time.Now().UnixNano()
	for _, region := range regions {
		n := &Node{
			timestamp: now,
			eventType: "Split",
			leader:    origin.leader,
			meta:      region,
			parents:   []int{index},
			children:  []int{},
		}
		h.nodes = append(h.nodes, n)

		// update origin info
		h.latest[region.GetId()] = len(h.nodes) - 1
		origin.children = append(origin.children, len(h.nodes)-1)
	}
}

func (h *regionHistory) onRegionMerge(region *core.RegionInfo, overlaps []*metapb.Region) {
	now := time.Now().UnixNano()
	var parents []int

	// regard origin region as overlap too
	overlaps = append(overlaps, region.GetMeta())

	// make sure in order
	sort.Slice(overlaps, func(i, j int) bool {
		return bytes.Compare(overlaps[i].GetStartKey(), overlaps[j].GetStartKey()) < 0
	})

	for _, overlap := range overlaps {
		index, ok := h.latest[overlap.GetId()]
		if !ok {
			log.Errorf("[Merge] not found latest info of region %v", overlap.GetId())
			return
		}
		parents = append(parents, index)
		// get the next index that following n will be
		// make sure there is no function appedn h.nodes concurrently
		h.nodes[index].children = append(h.nodes[index].children, len(h.nodes))
	}

	n := &Node{
		timestamp: now,
		eventType: "Merge",
		leader:    region.GetLeader().GetStoreId(),
		meta:      region.GetMeta(),
		parents:   parents,
		children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = len(h.nodes) - 1
}

func (h *regionHistory) onRegionLeaderChange(region *core.RegionInfo) {
	now := time.Now().UnixNano()

	index, ok := h.latest[region.GetID()]
	if !ok {
		log.Errorf("[Leader] not found latest info of region %v", region.GetID())
		return
	}
	origin := h.nodes[index]

	n := &Node{
		timestamp: now,
		eventType: "LeaderChange",
		leader:    region.GetLeader().GetStoreId(),
		meta:      region.GetMeta(),
		parents:   []int{index},
		children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = len(h.nodes) - 1
	origin.children = append(origin.children, len(h.nodes)-1)
}

func (h *regionHistory) onRegionConfChange(region *core.RegionInfo) {
	now := time.Now().UnixNano()

	index, ok := h.latest[region.GetID()]
	if !ok {
		log.Errorf("[Conf] not found latest info of region %v", region.GetID())
		return
	}
	origin := h.nodes[index]

	n := &Node{
		timestamp: now,
		eventType: "ConfChange",
		leader:    region.GetLeader().GetStoreId(),
		meta:      region.GetMeta(),
		parents:   []int{index},
		children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = len(h.nodes) - 1
	origin.children = append(origin.children, len(h.nodes)-1)
}

func (h *regionHistory) onRegionInsert(region *core.RegionInfo) {
	_, ok := h.latest[region.GetID()]
	if ok {
		// means it comes from split
		// we handle it in onRegionSplit, skip here
		return
	} else {
		// the first region
		now := time.Now().UnixNano()
		n := &Node{
			timestamp: now,
			eventType: "Init",
			leader:    region.GetLeader().GetStoreId(),
			meta:      region.GetMeta(),
			parents:   []int{},
			children:  []int{},
		}
		h.nodes = append(h.nodes, n)
		h.latest[region.GetID()] = len(h.nodes) - 1
	}
}

func (h *regionHistory) getHistoryList(start, end int64) []*Node {
	return nil
}

func (h *regionHistory) getRegionHistoryList(regionID uint64) []*Node {
	return nil
}

func (h *regionHistory) getKeyHistoryList(key []byte) []*Node {
	return nil
}
