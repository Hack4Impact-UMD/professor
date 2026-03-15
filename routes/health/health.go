package health

import (
	"encoding/json"
	"net/http"

	"github.com/Hack4Impact-UMD/professor/util"
)

type healthResponse struct {
	Status        string `json:"status"`
	BunAvailable  bool   `json:"bunAvailable"`
	NodeAvailable bool   `json:"nodeAvailable"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	bunExists := util.CommandExists("bun")
	nodeExists := util.CommandExists("node")

	if err := json.NewEncoder(w).Encode(healthResponse{
		Status:        "OK",
		BunAvailable:  bunExists,
		NodeAvailable: nodeExists,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", healthHandler)
}
