package health

import (
	"net/http"
	"os"

	"github.com/Hack4Impact-UMD/professor/util"
)

type healthResponse struct {
	Status        string `json:"status"`
	PnpmAvailable bool   `json:"pnpmAvailable"`
	NodeAvailable bool   `json:"nodeAvailable"`
	PatExists     bool   `json:"patExists"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	pnpmExists := util.CommandExists("pnpm")
	nodeExists := util.CommandExists("node")
	patExists := os.Getenv("GITHUB_PAT") != ""
	status := "DOWN"

	if pnpmExists && nodeExists {
		status = "OK"
	}

	util.JSON(w, healthResponse{
		Status:        status,
		PnpmAvailable: pnpmExists,
		NodeAvailable: nodeExists,
		PatExists:     patExists,
	})
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", healthHandler)
}
