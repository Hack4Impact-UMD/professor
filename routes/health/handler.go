package health

import (
	"github.com/Hack4Impact-UMD/professor/util"
	"net/http"
)

type healthResponse struct {
	Status        string `json:"status"`
	BunAvailable  bool   `json:"bunAvailable"`
	NodeAvailable bool   `json:"nodeAvailable"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	bunExists := util.CommandExists("bun")
	nodeExists := util.CommandExists("node")
	status := "DOWN"

	if bunExists && nodeExists {
		status = "OK"
	}

	util.JSON(w, healthResponse{
		Status:        status,
		BunAvailable:  bunExists,
		NodeAvailable: nodeExists,
	})
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", healthHandler)
}
