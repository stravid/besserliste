package main

import (
	"html/template"
	"net/http"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
)

func (env *Environment) HomeRoute(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct {
		CurrentUser types.User
	}{
		CurrentUser: user,
	}

	files := []string{
		"screens/home.html",
		"layouts/internal.html",
	}

	ts, err := template.ParseFS(web.Templates, files...)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	err = ts.Execute(w, data)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}
}
