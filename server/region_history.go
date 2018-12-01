package server

import (
	"bytes"
	"sort"
	"sync"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server/core"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	idx       int
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

func (n *Node) GetTimestamp() int64 {
	return n.timestamp
}

func (n *Node) GetLeader() uint64 {
	return n.leader
}

func (n *Node) GetEventType() string {
	return n.eventType
}

func (n *Node) GetMeta() *metapb.Region {
	return n.meta
}

func (n *Node) GetParents() []int {
	return n.parents
}

func (n *Node) GetChildren() []int {
	return n.children
}

type regionHistory struct {
	sync.RWMutex

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
	h.Lock()
	defer h.Unlock()

	index, ok := h.latest[originID]
	if !ok {
		log.Errorf("[Split] not found latest info of region %v", originID)
		return
	}
	origin := h.nodes[index]

	now := time.Now().UnixNano()
	for _, region := range regions {
		idx := len(h.nodes)
		n := &Node{
			idx:       idx,
			timestamp: now,
			eventType: "Split",
			leader:    origin.leader,
			meta:      region,
			parents:   []int{index},
			children:  []int{},
		}
		h.nodes = append(h.nodes, n)

		// update origin info
		h.latest[region.GetId()] = idx
		origin.children = append(origin.children, idx)
	}
}

func (h *regionHistory) onRegionMerge(region *core.RegionInfo, overlaps []*metapb.Region) {
	h.Lock()
	defer h.Unlock()

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
	idx := len(h.nodes)
	n := &Node{
		idx:       idx,
		timestamp: now,
		eventType: "Merge",
		leader:    region.GetLeader().GetStoreId(),
		meta:      region.GetMeta(),
		parents:   parents,
		children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = idx
}

func (h *regionHistory) onRegionLeaderChange(region *core.RegionInfo) {
	h.Lock()
	defer h.Unlock()

	now := time.Now().UnixNano()
	index, ok := h.latest[region.GetID()]
	if !ok {
		log.Errorf("[Leader] not found latest info of region %v", region.GetID())
		return
	}

	origin := h.nodes[index]
	idx := len(h.nodes)

	n := &Node{
		idx:       idx,
		timestamp: now,
		eventType: "LeaderChange",
		leader:    region.GetLeader().GetStoreId(),
		meta:      region.GetMeta(),
		parents:   []int{index},
		children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = idx
	origin.children = append(origin.children, idx)
}

func (h *regionHistory) onRegionConfChange(region *core.RegionInfo) {
	h.Lock()
	defer h.Unlock()

	now := time.Now().UnixNano()
	index, ok := h.latest[region.GetID()]
	if !ok {
		log.Errorf("[Conf] not found latest info of region %v", region.GetID())
		return
	}
	origin := h.nodes[index]
	idx := len(h.nodes)

	n := &Node{
		idx:       idx,
		timestamp: now,
		eventType: "ConfChange",
		leader:    region.GetLeader().GetStoreId(),
		meta:      region.GetMeta(),
		parents:   []int{index},
		children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = idx
	origin.children = append(origin.children, idx)
}

func (h *regionHistory) onRegionBootstrap(region *metapb.Region) {
	h.Lock()
	defer h.Unlock()

	log.Infof("[Boorstrap] region %v", region.GetId())
	idx := len(h.nodes)
	// the first region
	now := time.Now().UnixNano()
	n := &Node{
		idx:       idx,
		timestamp: now,
		eventType: "Bootstrap",
		leader:    0,
		meta:      region,
		parents:   []int{},
		children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	h.latest[region.GetId()] = idx
}

func (h *regionHistory) lower_bound(x int64) int {
	l := 0
	r := len(h.nodes)
	for l < r {
		mid := (l + r) >> 1
		if h.nodes[mid].timestamp == x {
			return mid
		} else if h.nodes[mid].timestamp <= x {
			l = mid
		} else {
			r = mid
		}
	}
	return r
}

func (h *regionHistory) getHistoryList(start, end int64) []*Node {
	h.RLock()
	defer h.RUnlock()

	if end < start {
		return nil
	}
	stIndex := h.lower_bound(start)
	edIndex := h.lower_bound(end)
	if edIndex < len(h.nodes) && h.nodes[edIndex].timestamp == end {
		edIndex++
	}
	if edIndex > stIndex {
		return h.nodes[stIndex:edIndex]
	}
	return nil
}

func (h *regionHistory) findPrevNodes(index int, start int64, end int64) []*Node {
	var que []int
	var ans []*Node
	que = append(que, index)
	mp := make(map[int]bool)
	st := 0
	ed := 1
	mp[index] = true
	for st < ed {
		v := que[st]
		st++
		if h.nodes[v].timestamp >= start && h.nodes[v].timestamp <= end {
			ans = append(ans, h.nodes[v])
		}
		for _, u := range h.nodes[v].parents {
			if _, ok := mp[u]; ok {
				continue
			}
			if h.nodes[u].timestamp < start {
				continue
			}
			que = append(que, u)
			mp[u] = true
			ed++
		}
	}
	return ans
}

func (h *regionHistory) filter(ans []*Node) []*Node {
	h.RLock()
	defer h.RUnlock()

	if len(ans) == 0 {
		return nil
	}
	nodes := make([]*Node, len(ans))
	mp := make(map[int]int)
	for i, n := range ans {
		mp[int(n.meta.GetId())] = i
	}
	for i, n := range ans {
		nodes[i] = &Node{
			idx:       n.idx,
			timestamp: n.timestamp,
			eventType: n.eventType,
			leader:    n.leader,
			meta:      n.meta,
			parents:   []int{},
			children:  []int{},
		}
		*nodes[i] = *n
		nodes[i].parents = make([]int, 0)
		nodes[i].children = make([]int, 0)
		for _, j := range n.parents {
			if k, ok := mp[j]; ok {
				nodes[i].parents = append(nodes[i].parents, k)
			}
		}
		for _, j := range n.children {
			if k, ok := mp[j]; ok {
				nodes[i].children = append(nodes[i].children, k)
			}
		}
	}
	return nodes
}

func (h *regionHistory) getRegionHistoryList(regionID uint64, start int64, end int64) []*Node {
	h.RLock()
	defer h.RUnlock()

	index, ok := h.latest[regionID]
	if ok {
		return h.filter(h.findPrevNodes(index, start, end))
	}
	h.RLock()
	defer h.RUnlock()

	return nil
}

func (h *regionHistory) getKeyHistoryList(key []byte, regionID uint64, start int64, end int64) []*Node {
	h.RLock()
	defer h.RUnlock()

	index, ok := h.latest[regionID]
	if !ok {
		return nil
	}
	ans := h.findPrevNodes(index, start, end)
	keyStr := string(key)
	var res []*Node
	for _, v := range ans {
		if v.meta.GetStartKey() != nil && string(v.meta.GetStartKey()) < keyStr {
			res = append(res, v)
		}
	}
	return h.filter(res)
}
