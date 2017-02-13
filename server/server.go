// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

var templatePath string

var templates *template.Template

func InitServer() {
	templatePath = "webapp" + string(os.PathSeparator) + "templates" + string(os.PathSeparator)
	osm, err := Asset(templatePath + "inputform.html")
	if err != nil {
		log.Fatal(err)
	}

	if templates, err = template.New("inputform.html").Parse(string(osm)); err != nil {
		log.Fatal(err)
	}

	/* HTML handlers */
	http.HandleFunc("/new/", newHandler)                /* Handler for input form */
	http.HandleFunc("/run/", runHandler)                /* Handler for running a new analysis */
	http.HandleFunc("/view/", makeHandler(viewHandler)) /* Handler for viewing analysis results */

	/* Static files handlers : js, css, etc. */
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(assetFS())))
	http.Handle("/", http.RedirectHandler("/new/", http.StatusFound))

	http.ListenAndServe(":8080", nil)
}
