package core

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
	"time"

	"math"

	"github.com/pingcap/kvproto/pkg/metapb"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	Idx       int
	Timestamp int64 // UnixNano-nano
	Leader    uint64
	EventType string

	Meta *metapb.Region

	// make sure the range of pointing regions are in ascend order
	// intS is index of nodes
	Parents  []int
	Children []int
}

func (n *Node) GetTimestamp() int64 {
	return n.Timestamp
}

func (n *Node) GetLeader() uint64 {
	return n.Leader
}

func (n *Node) GetEventType() string {
	return n.EventType
}

func (n *Node) GetMeta() *metapb.Region {
	return n.Meta
}

func (n *Node) GetParents() []int {
	return n.Parents
}

func (n *Node) GetChildren() []int {
	return n.Children
}

func (n *Node) IsKeyInRange(key string) bool {
	stKey := n.Meta.GetStartKey()
	if stKey != nil && string(stKey) > key {
		return false
	}
	edKey := n.Meta.GetEndKey()
	if edKey != nil && string(edKey) != "" && string(edKey) < key {
		return false
	}
	return true
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

func (h *RegionHistory) GetHistoryCount() int {
	return len(h.nodes)
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
	log.Infof("[Split] region %v, ts: %v", originID, now)
	origin := h.nodes[index]

	// make sure in order
	sort.Slice(regions, func(i, j int) bool {
		return bytes.Compare(regions[i].GetStartKey(), regions[j].GetStartKey()) < 0
	})

	for _, region := range regions {
		idx := len(h.nodes)
		n := &Node{
			Idx:       idx,
			Timestamp: now,
			EventType: "Split",
			Leader:    origin.Leader,
			Meta:      region,
			Parents:   []int{index},
			Children:  []int{},
		}
		h.nodes = append(h.nodes, n)
		if err := h.kv.SaveNode(n); err != nil {
			panic(fmt.Sprintf("[Split] unable to save", n.Meta.GetId()))
		}

		// update origin info
		h.latest[region.GetId()] = idx
		origin.Children = append(origin.Children, idx)
	}

	if err := h.kv.SaveNode(origin); err != nil {
		panic(fmt.Sprintf("[Split] unable to save", origin.Meta.GetId()))
	}
}

func (h *RegionHistory) OnRegionMerge(region *RegionInfo, overlaps []*metapb.Region) {
	h.Lock()
	defer h.Unlock()

	// some bug for tikv or pd, just skip it
	if region.GetID() == 2 && len(region.GetStartKey()) == 0 && len(region.GetEndKey()) == 0 {
		log.Infof("[Merge] skip region 2 bug")
		return
	}
	for _, overlap := range overlaps {
		if overlap.GetId() == 2 && len(overlap.GetStartKey()) == 0 && len(overlap.GetEndKey()) == 0 {
			if index, found := h.latest[2]; found && h.nodes[index].Meta.GetRegionEpoch().GetVersion() > overlap.GetRegionEpoch().GetVersion() {
				log.Infof("[Merge] skip region 2 bug")
				return
			}
		}
	}

	now := time.Now().UnixNano()
	log.Infof("[Merge] region %v, ts: %v", region.GetID(), now)
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
		// make sure there is no function append h.nodes concurrently
		h.nodes[index].Children = append(h.nodes[index].Children, len(h.nodes))
		if err := h.kv.SaveNode(h.nodes[index]); err != nil {
			panic(fmt.Sprintf("[Merge] unable to save", h.nodes[index].Meta.GetId()))
		}
	}

	idx := len(h.nodes)
	n := &Node{
		Idx:       idx,
		Timestamp: now,
		EventType: "Merge",
		Leader:    region.GetLeader().GetStoreId(),
		Meta:      region.GetMeta(),
		Parents:   parents,
		Children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = idx
	if err := h.kv.SaveNode(n); err != nil {
		panic(fmt.Sprintf("[Merge] unable to save", n.Meta.GetId()))
	}
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

	log.Infof("[Leader] region %v, ts: %v", region.GetID(), now)
	origin := h.nodes[index]
	idx := len(h.nodes)

	n := &Node{
		Idx:       idx,
		Timestamp: now,
		EventType: "LeaderChange",
		Leader:    region.GetLeader().GetStoreId(),
		Meta:      region.GetMeta(),
		Parents:   []int{index},
		Children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = idx
	if err := h.kv.SaveNode(n); err != nil {
		panic(fmt.Sprintf("[Leader] unable to save", n.Meta.GetId()))
	}
	origin.Children = append(origin.Children, idx)
	if err := h.kv.SaveNode(origin); err != nil {
		panic(fmt.Sprintf("[Leader] unable to save", origin.Meta.GetId()))
	}
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
		Idx:       idx,
		Timestamp: now,
		EventType: "ConfChange",
		Leader:    region.GetLeader().GetStoreId(),
		Meta:      region.GetMeta(),
		Parents:   []int{index},
		Children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	// update origin info
	h.latest[region.GetID()] = idx
	origin.Children = append(origin.Children, idx)
	if err := h.kv.SaveNode(n); err != nil {
		panic(fmt.Sprintf("[Conf] unable to save", n.Meta.GetId()))
	}
	if err := h.kv.SaveNode(origin); err != nil {
		panic(fmt.Sprintf("[Conf] unable to save", origin.Meta.GetId()))
	}
}

func (h *RegionHistory) OnRegionBootstrap(store *metapb.Store, region *metapb.Region) {
	h.Lock()
	defer h.Unlock()

	now := time.Now().UnixNano()
	log.Infof("[Bootstrap] region %v, ts: %v", region.GetId(), now)
	idx := len(h.nodes)
	// the first region
	n := &Node{
		Idx:       idx,
		Timestamp: now,
		EventType: "Bootstrap",
		Leader:    store.GetId(),
		Meta:      region,
		Parents:   []int{},
		Children:  []int{},
	}
	h.nodes = append(h.nodes, n)
	h.latest[region.GetId()] = idx
	if err := h.kv.SaveNode(n); err != nil {
		panic(fmt.Sprintf("[Bootstrap] unable to save", n.Meta.GetId()))
	}
}

func (h *RegionHistory) lower_bound(x int64) int {
	l := 0
	r := len(h.nodes)
	ans := r
	for l < r {
		mid := (l + r) >> 1
		if h.nodes[mid].Timestamp >= x {
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
	if edIndex < len(h.nodes) && h.nodes[edIndex].Timestamp == end {
		edIndex++
	}
	log.Infof("[getHistoryList] %v, stIndex: %v, edIndex : %v", len(h.nodes), stIndex, edIndex)
	if edIndex > stIndex {
		return h.filter(h.nodes[stIndex:edIndex], true)
	}
	return nil
}

func (h *RegionHistory) findKeyHistory(index int, key string, start int64, end int64) []*Node {
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
		if !h.nodes[v].IsKeyInRange(key) {
			continue
		}

		if h.nodes[v].Timestamp >= start && h.nodes[v].Timestamp <= end {
			ans = append(ans, h.nodes[v])
		}
		for _, u := range h.nodes[v].Parents {
			if _, ok := mp[u]; ok {
				continue
			}
			if h.nodes[u].Timestamp < start {
				continue
			}
			que = append(que, u)
			mp[u] = true
			ed++
		}

	}
	sort.Slice(ans, func(i, j int) bool { return ans[i].Idx < ans[j].Idx })
	log.Infof("[find key history] index: %d, count : %v", index, ed)
	return ans
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
		if h.nodes[v].Timestamp >= start && h.nodes[v].Timestamp <= end {
			ans = append(ans, h.nodes[v])
		}
		for _, u := range h.nodes[v].Children {
			if _, ok := mp[u]; ok {
				continue
			}
			if h.nodes[u].Timestamp < start || u >= index {
				continue
			}
			mp[u] = true
			ans = append(ans, h.nodes[u])
		}
		if h.nodes[v].Meta.GetId() != h.nodes[index].Meta.GetId() {
			continue
		}
		for _, u := range h.nodes[v].Parents {
			if _, ok := mp[u]; ok {
				continue
			}
			if h.nodes[u].Timestamp < start {
				continue
			}
			que = append(que, u)
			mp[u] = true
			ed++
		}

	}
	sort.Slice(ans, func(i, j int) bool { return ans[i].Idx < ans[j].Idx })
	log.Infof("[find prev node] index: %d, count : %v", index, ed)
	return ans
}

func (h *RegionHistory) filter(ans []*Node, addFinal bool) []*Node {
	h.RLock()
	defer h.RUnlock()

	log.Infof("[filter Region Begin] count : %v", len(ans))
	if len(ans) == 0 {
		return nil
	}
	nodes := make([]*Node, len(ans))
	mp := make(map[int]int)
	for i, n := range ans {
		mp[int(n.Idx)] = i
	}
	endTs := ans[len(ans)-1].Timestamp
	for i, n := range ans {
		nodes[i] = &Node{
			Idx:       n.Idx,
			Timestamp: n.Timestamp,
			EventType: n.EventType,
			Leader:    n.Leader,
			Meta:      n.Meta,
			Parents:   []int{},
			Children:  []int{},
		}
		for _, j := range n.Parents {
			if k, ok := mp[j]; ok {
				nodes[i].Parents = append(nodes[i].Parents, k)
			}
		}
		for _, j := range n.Children {
			if k, ok := mp[j]; ok {
				nodes[i].Children = append(nodes[i].Children, k)
			}
		}
		if len(nodes[i].Children) == 0 && nodes[i].Timestamp != endTs && addFinal {
			idx := len(nodes)
			nodes = append(nodes, &Node{
				Idx:       idx,
				Timestamp: endTs,
				Leader:    nodes[i].Leader,
				EventType: "Final",
				Meta:      nodes[i].Meta,
				Parents:   []int{i},
				Children:  []int{},
			})
			nodes[i].Children = append(nodes[i].Children, idx)
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
		return h.filter(h.findPrevNodes(index, start, end), false)
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
	keyStr := string(key)
	if !h.nodes[index].IsKeyInRange(keyStr) {
		panic(fmt.Sprintf("key [%v] is not in regionID[%v], [%v, %v)", key, regionID, h.nodes[index].Meta.GetStartKey(), h.nodes[index].Meta.GetEndKey()))
	}
	return h.filter(h.findKeyHistory(index, keyStr, start, end), false)

}
