package server

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"

	"github.com/fredericlemoine/gotree/upload"
	"github.com/fredericlemoine/sbsweb/io"
)

type ErrorInfo struct {
	Message string
}

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "text/html")
	if t, err2 := getTemplate("error"); err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
	} else {
		if err2 := t.ExecuteTemplate(w, "layout", ErrorInfo{err.Error()}); err2 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	var info interface{}
	if t, err := getTemplate("index"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		if err := t.ExecuteTemplate(w, "layout", info); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

	if t, err := getTemplate("view"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		if err := t.ExecuteTemplate(w, "layout", a); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
	if a.Status == STATUS_FINISHED || a.Status == STATUS_TIMEOUT {
		upld := upload.NewItolUploader("", "")
		url, _, err := upld.Upload(fmt.Sprintf("%d", a.Id), a.result)
		if err != nil {
			io.LogError(err)
			errorHandler(w, r, err)
			return
		}
		http.Redirect(w, r, url, http.StatusSeeOther)
		return
	}
	finishederr := errors.New("Analysis is not finished")
	io.LogError(finishederr)
	errorHandler(w, r, finishederr)
	return
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	var info interface{}

	if t, err := getTemplate("inputform"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		if err := t.ExecuteTemplate(w, "layout", info); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

func getTemplate(name string) (*template.Template, error) {
	t, ok := templates[name]
	if !ok {
		return nil, errors.New("No template named " + name)
	}
	return t, nil
}
