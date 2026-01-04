package main

import (
	"errors"
	"net/http"
)

func (env *Environment) LogoutRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		env.session.Destroy(r)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		respondWithErrorPage(w, http.StatusBadRequest, errors.New("Logout muss per `POST` Methode passieren."))
	}
}
