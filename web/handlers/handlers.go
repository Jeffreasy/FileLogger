package handlers

import (
	"html/template"
	"net/http"
)

func HomePage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/layout.html"))
	tmpl.Execute(w, nil)
}

func ScanPage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/layout.html"))
	tmpl.Execute(w, nil)
}

func ResultsPage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("web/templates/layout.html"))
	tmpl.Execute(w, nil)
}
