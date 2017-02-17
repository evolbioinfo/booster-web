package server

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"

	"github.com/fredericlemoine/gotree/upload"
	"github.com/fredericlemoine/sbsweb/io"
)

type errorInfo struct {
	Message string
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
	//nw := t.Newick()
	//w.Write([]byte(nw))

	a, ok := getAnalysis(id)
	if !ok {
		existerr := errors.New("Analysis does not exist")
		io.LogError(existerr)
		errorHandler(w, r, existerr)
		return
	}
	err := templates.ExecuteTemplate(w, "view.html", a)
	if err != nil {
		io.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func itolHandler(w http.ResponseWriter, r *http.Request, id string) {
	a, ok := getAnalysis(id)
	if !ok {
		existerr := errors.New("Analysis does not exist")
		io.LogError(existerr)
		errorHandler(w, r, existerr)
		return
	}
	upld := upload.NewItolUploader("", "")
	url, _, err := upld.Upload(fmt.Sprintf("%d", a.Id), a.result)
	if err != nil {
		io.LogError(err)
		errorHandler(w, r, err)
		return
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	var info interface{}
	err := templates.ExecuteTemplate(w, "inputform.html", info)
	if err != nil {
		io.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	parserr := r.ParseMultipartForm(32 << 20)
	if parserr != nil {
		io.LogError(parserr)
		errorHandler(w, r, parserr)
		return
	}

	reftree, refhandler, err := r.FormFile("reftree")
	if err != nil {
		io.LogError(err)
		errorHandler(w, r, err)
		return
	}

	boottree, boothandler, err2 := r.FormFile("boottrees")
	if err2 != nil {
		io.LogError(err2)
		errorHandler(w, r, err2)
		return
	}

	refreader, referr := GetFormFileReader(reftree, refhandler)
	if referr != nil {
		io.LogError(referr)
		errorHandler(w, r, referr)
		return
	}

	bootreader, booterr := GetFormFileReader(boottree, boothandler)
	if booterr != nil {
		io.LogError(booterr)
		errorHandler(w, r, booterr)
		return
	}

	a := newAnalysis(refreader, bootreader, reftree, boottree)
	http.Redirect(w, r, "/view/"+a.Id, http.StatusSeeOther)

}

var validPath = regexp.MustCompile("^/(view|itol)/([-a-zA-Z0-9]+)$")

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

/* Returns the opened file and a buffered reader (gzip or not) for the file */
func GetFormFileReader(f multipart.File, h *multipart.FileHeader) (*bufio.Reader, error) {
	var reader *bufio.Reader
	if strings.HasSuffix(h.Filename, ".gz") {
		if gr, err := gzip.NewReader(f); err != nil {
			return nil, err
		} else {
			reader = bufio.NewReader(gr)
		}
	} else {
		reader = bufio.NewReader(f)
	}
	return reader, nil
}
