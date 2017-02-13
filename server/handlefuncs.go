package server

import (
	"net/http"
	"os"
	"regexp"

	"github.com/fredericlemoine/gotree/support"
)

type errorInfo struct {
	message string
}

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "text/html")
	err2 := templates.ExecuteTemplate(w, "error.html", errorInfo{err.Error()})
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Content-Type", "text/html")
	var info interface{}
	err := templates.ExecuteTemplate(w, "view.html", info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	var info interface{}
	err := templates.ExecuteTemplate(w, "inputform.html", info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	reftree, handler, err := r.FormFile("reftree")
	if err != nil {
		errorHandler(w, r, err)
		return
	}
	defer reftree.Close()

	boottree, handler2, err2 := r.FormFile("boottrees")
	if err2 != nil {
		errorHandler(w, r, err2)
		return
	}
	defer boottree.Close()
	w.Header().Set("Content-Type", "text/plain")
	t := support.MastLikeFile(reftree.Reader, boottree.Reader, os.Stderr, false, 1)
	nw := t.Newick()
	w.Write([]byte(nw))

	defer boottree.Close()
}

var validPath = regexp.MustCompile("^/(view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}
