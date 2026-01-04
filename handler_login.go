package main

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
)

func (env *Environment) LoginRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			respondWithErrorPage(w, http.StatusBadRequest, err)
			return
		}

		id, err := types.UserIdFromString(r.PostForm.Get("user_id"))
		if err != nil {
			respondWithErrorPage(w, http.StatusBadRequest, err)
			return
		}

		tx, err := env.db.Begin()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		user, err := env.queries.GetUserById(tx, *id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				respondWithErrorPage(w, http.StatusNotFound, err)
			} else {
				respondWithErrorPage(w, http.StatusInternalServerError, err)
				return
			}
		}

		err = tx.Commit()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		env.session.Put(r, "user_id", user.Id)

		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	} else {
		tx, err := env.db.Begin()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		users, err := env.queries.GetUsers(tx)
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		err = tx.Commit()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		files := []string{
			"screens/identify.html",
			"layouts/external.html",
		}

		ts, err := template.ParseFS(web.Templates, files...)
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		err = ts.Execute(w, users)
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
	}
}
