package grade

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
		slog.Default().Warn("bad request: failed to decode body", "err", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if gradeReq.JobId == "" || gradeReq.RepoURL == "" || gradeReq.TestRepo == "" {
		slog.Default().Warn("bad request: missing required fields", "jobId", gradeReq.JobId, "repoURL", gradeReq.RepoURL, "testRepo", gradeReq.TestRepo)
		http.Error(w, "jobId, repoURL, and testRepo are required", http.StatusBadRequest)
		return
	}

	slog.Default().Info("grade request received", "jobId", gradeReq.JobId, "repoURL", gradeReq.RepoURL, "testRepo", gradeReq.TestRepo)

	rep, err := reporter.NewFirestoreReporter(fsClient)
	if err != nil {
		slog.Default().Error("reporter creation failed", "jobId", gradeReq.JobId, "err", err)
		http.Error(w, fmt.Sprintf("reporter create error: %v", err.Error()), http.StatusInternalServerError)
		return
	}
	if err := RunGradingJob(gradeReq.JobId, gradeReq.RepoURL, gradeReq.TestRepo, rep, slog.Default()); err != nil {
		slog.Default().Error("grading job failed", "jobId", gradeReq.JobId, "err", err)
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
