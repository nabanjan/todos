package main

import (
	"html/template"
	"net/http"
)

// ContactDetails ...
type ContactDetails struct {
	Email   string
	Subject string
	Message string
}

func mainForms() {
	tmpl := template.Must(template.ParseFiles("static/templates/forms.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			tmpl.Execute(w, nil)
			return
		}

		details := ContactDetails{
			Email:   r.FormValue("email"),
			Subject: r.FormValue("subject"),
			Message: r.FormValue("message"),
		}

		// TODO: do something with details
		_ = details

		tmpl.Execute(w, struct{ Success bool }{true})
	})

	http.ListenAndServe(":8080", nil)
}
