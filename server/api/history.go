package api

import (
	"github.com/pingcap/pd/server"
	"github.com/unrolled/render"
)

type historyHandler struct {
	svr *server.Server
	rd  *render.Render
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
		rd:  rd,
	}
}
