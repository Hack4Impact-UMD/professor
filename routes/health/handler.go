package health

import (
	"github.com/Hack4Impact-UMD/professor/util"
	"net/http"
)

type healthResponse struct {
	Status        string `json:"status"`
	PnpmAvailable bool   `json:"pnpmAvailable"`
	NodeAvailable bool   `json:"nodeAvailable"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	pnpmExists := util.CommandExists("pnpm")
	nodeExists := util.CommandExists("node")
	status := "DOWN"

	if pnpmExists && nodeExists {
		status = "OK"
	}

	util.JSON(w, healthResponse{
		Status:        status,
		PnpmAvailable: pnpmExists,
		NodeAvailable: nodeExists,
	})
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", healthHandler)
}
