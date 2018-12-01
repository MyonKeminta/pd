package api

import (
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server"
	"github.com/unrolled/render"
)

type historyHandler struct {
	svr *server.Server
	r   *render.Render
}

func newHistoryHandler(svr *server.Server, rd *render.Render) *historyHandler {
	return &historyHandler{
		svr: svr,
		r:   rd,
	}
}

type NodeInfo struct {
	timestamp int64       `json:"timestamp`
	leader    uint64      `json:"leader_store_id"`
	eventType string      `json:"event_type"`
	meta      *RegionMeta `json:"region"`
	parents   []int       `json:"parents"`
	children  []int       `json:"children"`
}

type RegionMeta struct {
	id          uint64              `json:"id"`
	startKey    string              `json:"start_key"`
	endKey      string              `json:"end_key"`
	regionEpoch *metapb.RegionEpoch `json:"region_epoch"`
	peers       []*metapb.Peer      `json:"peers"`
}

// type EpochInfo struct {
// 	id  	uint64 `json:"id"`
// 	storeID 	uint64 `json:"store_id"`
// }

func newNodeInfo(node *server.Node) *NodeInfo {
	return &NodeInfo{
		timestamp: node.GetTimestamp(),
		leader:    node.GetLeader(),
		meta:      newRegionMeta(node.GetMeta()),
		eventType: node.GetEventType(),
		parents:   node.GetParents(),
		children:  node.GetChildren(),
	}
}

func newRegionMeta(meta *metapb.Region) *RegionMeta {
	return &RegionMeta{
		id:          meta.Id,
		startKey:    hex.EncodeToString(meta.StartKey),
		endKey:      hex.EncodeToString(meta.EndKey),
		regionEpoch: meta.RegionEpoch,
		peers:       meta.Peers,
	}
}

func convertToAPINodeInfos(nodes []*server.Node) []*NodeInfo {
	list := make([]*NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		list = append(list, newNodeInfo(node))
	}
	return list
}

func (h *historyHandler) List(w http.ResponseWriter, r *http.Request) {
	cluster := h.svr.GetRaftCluster()
	if cluster == nil {
		h.r.JSON(w, http.StatusInternalServerError, server.ErrNotBootstrapped.Error())
		return
	}

	startStr := r.URL.Query().Get("start")
	if startStr == "" {
		startStr = "0"
	}
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	endStr := r.URL.Query().Get("end")
	if endStr == "" {
		endStr = "0"
	}
	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	list := cluster.GetHistoryList(start, end)

	h.r.JSON(w, http.StatusOK, convertToAPINodeInfos(list))
}

func (h *historyHandler) Region(w http.ResponseWriter, r *http.Request) {
	cluster := h.svr.GetRaftCluster()
	if cluster == nil {
		h.r.JSON(w, http.StatusInternalServerError, server.ErrNotBootstrapped.Error())
		return
	}

	startStr := r.URL.Query().Get("start")
	if startStr == "" {
		startStr = "0"
	}
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	endStr := r.URL.Query().Get("end")
	if endStr == "" {
		endStr = "0"
	}
	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	regionID, err := strconv.ParseUint(mux.Vars(r)["region_id"], 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	list := cluster.GetRegionHistoryList(start, end, regionID)

	h.r.JSON(w, http.StatusOK, convertToAPINodeInfos(list))
}

func (h *historyHandler) Key(w http.ResponseWriter, r *http.Request) {
	cluster := h.svr.GetRaftCluster()
	if cluster == nil {
		h.r.JSON(w, http.StatusInternalServerError, server.ErrNotBootstrapped.Error())
		return
	}

	startStr := r.URL.Query().Get("start")
	if startStr == "" {
		startStr = "0"
	}
	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	endStr := r.URL.Query().Get("end")
	if endStr == "" {
		endStr = "0"
	}
	end, err := strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	key, err := hex.DecodeString(mux.Vars(r)["key"])
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	list := cluster.GetKeyHistoryList(start, end, key)

	h.r.JSON(w, http.StatusOK, convertToAPINodeInfos(list))
}
