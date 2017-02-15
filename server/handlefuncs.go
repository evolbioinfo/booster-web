package server

import (
	"bufio"
	"compress/gzip"
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"

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
	hc := http.Client{}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("treeFile", "tree.nh")
	if err != nil {
		io.LogError(err)
		errorHandler(w, r, err)
		return
	}
	part.Write(a.result.Newick())
	_ = writer.WriteField("treeFormat", "newick")
	_ = writer.WriteField("treeName", a.Id)

	err = writer.Close()
	if err != nil {
		io.LogError(err)
		errorHandler(w, r, err)
		return
	}
	req, err2 := http.NewRequest("POST", "http://itol.embl.de/batch_uploader.cgi", body)
	if err2 != nil {
	}
	resp, err3 := client.Do(request)
	if err3 != nil {
		io.LogError(err3)
		errorHandler(w, r, err3)
		return
	}
	defer resp.Body.Close()
	bodyresp, err4 := ioutil.ReadAll(resp.Body)
	if err4 != nil {
		io.LogError(err4)
		errorHandler(w, r, err4)
		return
	}
	stringresp := string(bodyresp)
	infos := strings.Split(stringresp, "\n")
	// succregexp := regexp.Compile("^ERR")
	// errregexp := regexp.Compile("^SUCCESS: (\\S+)")
	// if errregexp.MatchString(infos[len(infos)-1]) {
	// 	io.LogError(errors.New(fmt.Sprintf("Upload failed. iTOL returned the following error message: %s", infos[len(infos)-1])))
	// 	errorHandler(w, r, err4)
	// 	return
	// }
	// if succregexp.MatchString(infos[len(infos)-1]) {
	// 	io.LogError(errors.New(fmt.Sprintf("Upload failed. iTOL returned the following error message: %s", infos[len(infos)-1])))
	// 	errorHandler(w, r, err4)
	// 	return
	// }
	w.Header().Set("Content-Type", "text/plain")
	w.WriteString(stringresp)
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

var validPath = regexp.MustCompile("^/(view)/([-a-zA-Z0-9]+)$")

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
