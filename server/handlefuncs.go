/*

BOOSTER-WEB: Web interface to BOOSTER (https://github.com/evolbioinfo/booster)
Alternative method to compute bootstrap branch supports in large trees.

Copyright (C) 2017 BOOSTER-WEB dev team

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

*/

package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/fredericlemoine/booster-web/io"
	"github.com/fredericlemoine/booster-web/model"
	"github.com/fredericlemoine/booster-web/templates"
	"github.com/fredericlemoine/gotree/draw"
	"github.com/fredericlemoine/gotree/io/newick"
	"github.com/fredericlemoine/gotree/upload"
)

type ErrorInfo struct {
	Message string
}

type MarkDownPage struct {
	Md string
}

type GenericResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Global informations about server given to different templates
type GlobalInformation struct {
	GalaxyProcessor   bool
	EmailNotification bool
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

func loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	var info interface{}
	if t, err := getTemplate("login"); err != nil {
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

// if rawsupports: Then the tree with raw distances and branch ids is uploaded to itol
// else the normalized support tree is upploaded.
func itolHandler(w http.ResponseWriter, r *http.Request, id string, rawdistances bool, fbptree bool) {
	a, err := getAnalysis(id)
	if err != nil {
		io.LogError(err)
		errorHandler(w, r, err)
		return
	}
	if a.Status == model.STATUS_FINISHED || a.Status == model.STATUS_TIMEOUT {
		upld := upload.NewItolUploader("", "")
		var uptree string
		if fbptree {
			uptree = a.FbpTree
		} else if rawdistances {
			uptree = a.TbeRawTree
		} else {
			uptree = a.TbeNormTree
		}
		t, err := newick.NewParser(strings.NewReader(uptree)).Parse()
		if err == nil {
			t.ClearPvalues()
			url, _, err := upld.UploadNewick(a.Id, t.Newick())
			if err != nil {
				io.LogError(err)
				errorHandler(w, r, err)
				return
			}
			http.Redirect(w, r, url, http.StatusSeeOther)
			return
		} else {
			io.LogError(err)
			errorHandler(w, r, err)
			return
		}
	}
	finishederr := errors.New("Analysis is not finished")
	io.LogError(finishederr)
	errorHandler(w, r, finishederr)
	return
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	info := GlobalInformation{
		GalaxyProcessor:   galaxyprocessor,
		EmailNotification: emailnotification,
	}

	if t, err := getTemplate("inputform"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		if err := t.ExecuteTemplate(w, "layout", info); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	var refseqs multipart.File
	var refseqshandler *multipart.FileHeader
	var reftree multipart.File
	var refhandler *multipart.FileHeader
	var boottree multipart.File
	var boothandler *multipart.FileHeader
	var err error
	var a *model.Analysis
	var nbootint int64
	var nbootrep string
	var workflow string
	var email string

	parserr := r.ParseMultipartForm(32 << 20)
	if parserr != nil {
		io.LogError(parserr)
		errorHandler(w, r, parserr)
		return
	}

	if refseqs, refseqshandler, err = r.FormFile("refseqs"); err != nil {
		err = errors.New("No Sequence file given: " + err.Error())
		io.LogInfo(err.Error())

		// No given sequence file
		// Then we take tree files
		if reftree, refhandler, err = r.FormFile("reftree"); err != nil {
			err = errors.New("No reference tree file given (nor sequence file): " + err.Error())
			io.LogError(err)
			errorHandler(w, r, err)
			return
		}
		defer reftree.Close()

		if boottree, boothandler, err = r.FormFile("boottrees"); err != nil {
			err = errors.New("No bootstrap tree file given (nor sequence file): " + err.Error())
			io.LogError(err)
			errorHandler(w, r, err)
			return
		}
		defer boottree.Close()
	}
	email = r.FormValue("email")

	workflow = r.FormValue("workflow")

	nbootrep = r.FormValue("nboot")
	if nbootint, err = strconv.ParseInt(nbootrep, 10, 64); err != nil && galaxyprocessor {
		io.LogError(err)
		errorHandler(w, r, err)
		return
	}
	if nbootint > 1000 {
		nbootint = 1000
	}

	if a, err = newAnalysis(refseqs, refseqshandler, reftree, refhandler, boottree, boothandler, email, int(nbootint), workflow); err != nil {
		err = errors.New("Error while creating a new analysis: " + err.Error())
		io.LogError(err)
		errorHandler(w, r, err)
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/view/"+a.Id, http.StatusSeeOther)
}

func apiAnalysisHandler(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Content-Type", "application/json")
	var a *model.Analysis
	a, err := getAnalysis(id)
	if err != nil {
		a = model.NewAnalysis()
		a.Message = err.Error()
		io.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(a)
}

func apiImageHandler(w http.ResponseWriter, r *http.Request, id string, collapse float64, layout, format string) {
	var a *model.Analysis
	a, err := getAnalysis(id)
	if err != nil {
		a = model.NewAnalysis()
		a.Message = err.Error()
		io.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if a.Status != model.STATUS_FINISHED {
		e := errors.New(fmt.Sprintf("Cannot draw image for a non finished analysis, status : %s", a.StatusStr()))
		io.LogError(e)
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}
	if a.TbeNormTree == "" {
		e := errors.New("Cannot draw image for an empty resulting tree")
		io.LogError(e)
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}

	t, err := newick.NewParser(strings.NewReader(a.TbeNormTree)).Parse()
	if err != nil {
		io.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		var d draw.TreeDrawer
		var l draw.TreeLayout
		encoder := base64.NewEncoder(base64.StdEncoding, w)

		switch format {
		case "svg":
			w.Header().Set("Content-Type", "image/svg+xml")
			d = draw.NewSvgTreeDrawer(w, 800, 800, 30, 30, 30, 30)
		case "png":
			w.Header().Set("Content-Type", "image/png;base64")
			d = draw.NewPngTreeDrawer(encoder, 800, 800, 30, 30, 30, 30)
		default:
			err := errors.New("Image format not recognized")
			io.LogError(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		switch layout {
		case "radial":
			l = draw.NewRadialLayout(d, false, false, false, true)
		case "circular":
			l = draw.NewCircularLayout(d, false, false, false, true)
		case "normal":
			l = draw.NewNormalLayout(d, false, false, false, true)
		default:
			err := errors.New("Tree layout not recognized")
			io.LogError(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !t.Rooted() {
			deepest := t.DeepestNode()
			t.Reroot(deepest)
		}
		l.SetSupportCutoff(collapse / 100.0)
		l.DrawTree(t)
		encoder.Close()
	}
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

var validRawNormPath = regexp.MustCompile("^/(view|itol)/([-a-zA-Z0-9]+)/(true|false)/(true|false)$")

//	* string: Analysis ID
//	* bool: If raw tree of normalized tree should be retrieved
func makeRawNormHandler(fn func(http.ResponseWriter, *http.Request, string, bool, bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validRawNormPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		rawdist, _ := strconv.ParseBool(m[3])
		fbptree, _ := strconv.ParseBool(m[4])
		fn(w, r, m[2], rawdist, fbptree)
	}
}

var validApiPath = regexp.MustCompile("^/api/(analysis)/([-a-zA-Z0-9]+)$")

func makeApiHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validApiPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

// URL of the form:
// /api/image/analysisid/bootstrapcutoff/treelayout/imageformat
var validApiImagePath = regexp.MustCompile("^/api/image/([-a-zA-Z0-9]+)/([0-9]+)/(circular|radial|normal)/(svg|png)$")

func makeApiImageHandler(fn func(http.ResponseWriter, *http.Request, string, float64, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validApiImagePath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		f, _ := strconv.ParseFloat(m[2], 64)
		fn(w, r, m[1], f, m[3], m[4])
	}
}

func getTemplate(name string) (*template.Template, error) {
	t, ok := templatesMap[name]
	if !ok {
		return nil, errors.New("No template named " + name)
	}
	return t, nil
}

func apiError(res http.ResponseWriter, err error) {
	answer := GenericResponse{
		1,
		err.Error(),
	}
	if err := json.NewEncoder(res).Encode(answer); err != nil {
		io.LogError(err)
	}
}
