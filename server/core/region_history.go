package core

import (
	"bytes"
	"sort"
	"sync"
	"time"

	"math"

	"github.com/pingcap/kvproto/pkg/metapb"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	idx       int
	timestamp int64 // UnixNano-nano
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

type RegionHistory struct {
	sync.RWMutex

	nodes []*Node
	kv    *KV

	// region_id -> index of nodes
	latest map[uint64]int
}

func NewRegionHistory(kv *KV) *RegionHistory {
	return &RegionHistory{
		kv:     kv,
		nodes:  make([]*Node, 0),
		latest: make(map[uint64]int),
	}
}

func (h *RegionHistory) OnRegionSplit(originID uint64, regions []*metapb.Region) {
	h.Lock()
	defer h.Unlock()

	index, ok := h.latest[originID]
	if !ok {
		log.Errorf("[Split] not found latest info of region %v", originID)
		return
	}
	now := time.Now().UnixNano()
	log.Infof("[RegionSplit] region %v, ts: %v", originID, now)
	origin := h.nodes[index]

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

func (h *RegionHistory) OnRegionMerge(region *RegionInfo, overlaps []*metapb.Region) {
	h.Lock()
	defer h.Unlock()

	now := time.Now().UnixNano()
	log.Infof("[RegionMerge] region %v, ts: %v", region.GetID(), now)
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

func (h *RegionHistory) OnRegionLeaderChange(region *RegionInfo) {
	h.Lock()
	defer h.Unlock()

	now := time.Now().UnixNano()
	index, ok := h.latest[region.GetID()]
	if !ok {
		log.Errorf("[Leader] not found latest info of region %v", region.GetID())
		return
	}

	log.Infof("[ConfChange] region %v, ts: %v", region.GetID(), now)
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

func (h *RegionHistory) OnRegionConfChange(region *RegionInfo) {
	h.Lock()
	defer h.Unlock()

	now := time.Now().UnixNano()
	index, ok := h.latest[region.GetID()]
	if !ok {
		log.Errorf("[Conf] not found latest info of region %v", region.GetID())
		return
	}
	log.Infof("[ConfChange] region %v, ts: %v", region.GetID(), now)
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

func (h *RegionHistory) OnRegionBootstrap(store *metapb.Store, region *metapb.Region) {
	h.Lock()
	defer h.Unlock()

	now := time.Now().UnixNano()
	log.Infof("[Bootstrap] region %v, ts: %v", region.GetId(), now)
	idx := len(h.nodes)
	// the first region
	n := &Node{
		idx:       idx,
		timestamp: now,
		eventType: "Bootstrap",
		leader:    store.GetId(),
		meta:      region,
		parents:   []int{},
		children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	h.latest[region.GetId()] = idx
}

func (h *RegionHistory) lower_bound(x int64) int {
	l := 0
	r := len(h.nodes)
	ans := r
	for l < r {
		mid := (l + r) >> 1
		if h.nodes[mid].timestamp >= x {
			ans = mid
			r = mid
		} else {
			l = mid + 1
		}
	}
	return ans
}

func (h *RegionHistory) GetHistoryList(start, end int64) []*Node {
	h.RLock()
	defer h.RUnlock()

	if end == 0 {
		end = math.MaxInt64
	}
	if end < start {
		log.Errorf("[getHistoryList ERROR] start: %v, end : %v", start, end)
		return nil
	}
	stIndex := h.lower_bound(start)
	edIndex := h.lower_bound(end)
	if edIndex < len(h.nodes) && h.nodes[edIndex].timestamp == end {
		edIndex++
	}
	log.Infof("[getHistoryList] %v, stIndex: %v, edIndex : %v", len(h.nodes), stIndex, edIndex)
	if edIndex > stIndex {
		return h.filter(h.nodes[stIndex:edIndex])
	}
	return nil
}

func (h *RegionHistory) findPrevNodes(index int, start int64, end int64) []*Node {
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
	log.Infof("[find prev node] index: %d, count : %v", index, ed)
	return ans
}

func (h *RegionHistory) filter(ans []*Node) []*Node {
	h.RLock()
	defer h.RUnlock()

	log.Infof("[filter Region Begin] count : %v", len(ans))
	if len(ans) == 0 {
		return nil
	}
	nodes := make([]*Node, len(ans))
	mp := make(map[int]int)
	for i, n := range ans {
		mp[int(n.idx)] = i
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
	log.Infof("[filter Region After] count : %v", len(nodes))
	return nodes
}

func (h *RegionHistory) GetRegionHistoryList(regionID uint64, start int64, end int64) []*Node {
	h.RLock()
	defer h.RUnlock()

	if end == 0 {
		end = math.MaxInt64
	}
	index, ok := h.latest[regionID]
	if ok {
		return h.filter(h.findPrevNodes(index, start, end))
	}

	return nil
}

func (h *RegionHistory) GetKeyHistoryList(key []byte, regionID uint64, start int64, end int64) []*Node {
	h.RLock()
	defer h.RUnlock()

	if end == 0 {
		end = math.MaxInt64
	}
	index, ok := h.latest[regionID]
	if !ok {
		return nil
	}
	ans := h.findPrevNodes(index, start, end)
	keyStr := string(key)
	var res []*Node
	for _, v := range ans {
		if v.meta.GetStartKey() == nil || string(v.meta.GetStartKey()) > keyStr {
			continue
		}
		if v.meta.GetEndKey() == nil || string(v.meta.GetEndKey()) < keyStr {
			continue
		}
		res = append(res, v)
	}
	return h.filter(res)
}
