package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/fredericlemoine/booster-web/io"
	"github.com/fredericlemoine/booster-web/model"
	"github.com/fredericlemoine/booster-web/templates"
	"github.com/fredericlemoine/gotree/io/newick"
	"github.com/fredericlemoine/gotree/upload"
)

type ErrorInfo struct {
	Message string
}

type MarkDownPage struct {
	Md string
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

func helpHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	helpmd, err := templates.Asset(templatePath + "help.md")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t, err := getTemplate("help"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		md := MarkDownPage{string(helpmd)}
		if err := t.ExecuteTemplate(w, "layout", md); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Content-Type", "text/html")
	//nw := t.Newick()
	//w.Write([]byte(nw))

	a, err := getAnalysis(id)
	if err != nil {
		io.LogError(err)
		errorHandler(w, r, err)
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
	a, err := getAnalysis(id)
	if err != nil {
		io.LogError(err)
		errorHandler(w, r, err)
		return
	}
	if a.Status == model.STATUS_FINISHED || a.Status == model.STATUS_TIMEOUT {
		upld := upload.NewItolUploader("", "")
		t, err := newick.NewParser(strings.NewReader(a.Result)).Parse()
		if err == nil {
			t.ClearPvalues()
			a.Result = t.Newick()
		}

		url, _, err := upld.UploadNewick(fmt.Sprintf("%d", a.Id), a.Result)
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

	a, err := newAnalysis(reftree, refhandler, boottree, boothandler)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	reftree.Close()
	boottree.Close()

	http.Redirect(w, r, "/view/"+a.Id, http.StatusSeeOther)
}

// 0<=Collapse<=100
func apiAnalysisHandler(w http.ResponseWriter, r *http.Request, id string, collapse float64) {
	w.Header().Set("Content-Type", "application/json")
	var a *model.Analysis
	a, err := getAnalysis(id)
	if err != nil {
		a = &model.Analysis{"none",
			"",
			"",
			"",
			model.STATUS_NOT_EXISTS,
			model.StatusStr(model.STATUS_NOT_EXISTS),
			err.Error(),
			0,
			"",
			"",
			"",
			"",
		}
		io.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t, err := newick.NewParser(strings.NewReader(a.Result)).Parse()
	if err == nil {
		/* We collapse lowly supported branches */
		t.ClearPvalues()
		a.Result = t.Newick()
		if collapse > 0 {
			t.CollapseLowSupport(collapse / 100)
		}
		a.Collapsed = t.Newick()
	} else {
		a.Message = "Cannot collapse branches : " + err.Error()
	}
	json.NewEncoder(w).Encode(a)
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

var validApiPath = regexp.MustCompile("^/api/(analysis)/([-a-zA-Z0-9]+)/([0-9]+)$")

func makeApiHandler(fn func(http.ResponseWriter, *http.Request, string, float64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validApiPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		f, _ := strconv.ParseFloat(m[3], 64)
		fn(w, r, m[2], f)
	}
}

func getTemplate(name string) (*template.Template, error) {
	t, ok := templatesMap[name]
	if !ok {
		return nil, errors.New("No template named " + name)
	}
	return t, nil
}
