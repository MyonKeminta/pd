package api

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pingcap/pd/server"
	"github.com/unrolled/render"
)

type historyHandler struct {
	svr *server.Server
	r   *render.Render
}

// history reflects the cluster's history.
// type history struct {
// 	Name       string   `json:"name"`
// 	MemberID   uint64   `json:"member_id"`
// 	ClientUrls []string `json:"client_urls"`
// 	history    bool     `json:"history"`
// }

func newHistoryHandler(svr *server.Server, rd *render.Render) *historyHandler {
	return &historyHandler{
		svr: svr,
		r:   rd,
	}
}

type NodeInfo struct {
}

func newNodeInfo(node *server.Node) {

}

func (h *historyHandler) List(w http.ResponseWriter, r *http.Request) {
	cluster := h.svr.GetRaftCluster()
	if cluster == nil {
		h.r.JSON(w, http.StatusInternalServerError, server.ErrNotBootstrapped.Error())
		return
	}

	start, err := strconv.ParseInt(mux.Vars(r)["start"], 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}
	end, err := strconv.ParseInt(mux.Vars(r)["end"], 10, 64)
	if err != nil {
		h.r.JSON(w, http.StatusBadRequest, err.Error())
		return
	}

	list := cluster.GetHistoryList(start, end)

	h.r.JSON(w, http.StatusOK, list)
}

func (h *historyHandler) Region(w http.ResponseWriter, r *http.Request) {

}

func (h *historyHandler) Key(w http.ResponseWriter, r *http.Request) {

}
