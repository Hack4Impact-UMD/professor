package grade

import (
	"encoding/json"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/Hack4Impact-UMD/professor/reporter"
)

type GradeRequest struct {
	JobId      string `json:"jobId"`
	ResponseId string `json:"responseId"`
	RepoURL    string `json:"repoURL"`
	TestRepo   string `json:"testRepo"`
}

func GradeHandler(w http.ResponseWriter, r *http.Request, fsClient *firestore.Client) {
	var gradeReq GradeRequest
	if err := json.NewDecoder(r.Body).Decode(&gradeReq); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if gradeReq.JobId == "" || gradeReq.RepoURL == "" || gradeReq.TestRepo == "" {
		http.Error(w, "jobId, repoURL, and testRepo are required", http.StatusBadRequest)
		return
	}

	rep, err := reporter.NewFirestoreReporter(fsClient)
	if err != nil {
		http.Error(w, fmt.Sprintf("reporter create error: %v", err.Error()), http.StatusInternalServerError)
		return
	}
	if err := RunGradingJob(gradeReq.JobId, gradeReq.RepoURL, gradeReq.TestRepo, rep); err != nil {
		http.Error(w, fmt.Sprintf("grading error: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "OK",
	})
}

func RegisterHandlers(mux *http.ServeMux, fsClient *firestore.Client) {
	mux.HandleFunc("POST /grade", func(w http.ResponseWriter, r *http.Request) {
		GradeHandler(w, r, fsClient)
	})
}
