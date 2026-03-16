package grade

import "net/http"

func GradeHandler(w http.ResponseWriter, r *http.Request) {

}

func RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("POST /grade", GradeHandler)
}
