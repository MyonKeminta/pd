package api

import (
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/pd/server"
	"github.com/pingcap/pd/server/core"
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
	Timestamp int64       `json:"timestamp"`
	Leader    uint64      `json:"leader_store_id"`
	EventType string      `json:"event_type"`
	Meta      *RegionMeta `json:"region"`
	Parents   []int       `json:"parents"`
	Children  []int       `json:"children"`
}

type RegionMeta struct {
	Id          uint64              `json:"id"`
	StartKey    string              `json:"start_key"`
	EndKey      string              `json:"end_key"`
	RegionEpoch *metapb.RegionEpoch `json:"region_epoch"`
	Peers       []*metapb.Peer      `json:"peers"`
}

func newNodeInfo(node *core.Node) *NodeInfo {
	return &NodeInfo{
		Timestamp: node.GetTimestamp(),
		Leader:    node.GetLeader(),
		EventType: node.GetEventType(),
		Meta:      newRegionMeta(node.GetMeta()),
		Parents:   node.GetParents(),
		Children:  node.GetChildren(),
	}
}

func newRegionMeta(meta *metapb.Region) *RegionMeta {
	return &RegionMeta{
		Id:          meta.Id,
		StartKey:    hex.EncodeToString(meta.StartKey),
		EndKey:      hex.EncodeToString(meta.EndKey),
		RegionEpoch: meta.RegionEpoch,
		Peers:       meta.Peers,
	}
}

func convertToAPINodeInfos(nodes []*core.Node) []*NodeInfo {
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
	regionID, err := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
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
