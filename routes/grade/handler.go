package grade

import (
	"net/http"

	"cloud.google.com/go/firestore"
)

func GradeHandler(w http.ResponseWriter, r *http.Request, fsClient *firestore.Client) {

}

func RegisterHandlers(mux *http.ServeMux, fsClient *firestore.Client) {
	mux.HandleFunc("POST /grade", func(w http.ResponseWriter, r *http.Request) {
		GradeHandler(w, r, fsClient)
	})
}
