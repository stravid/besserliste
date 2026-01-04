package main

import (
	"html/template"
	"net/http"
	"stravid.com/besserliste/web"
	"strconv"
)

func (env *Environment) LoginRoute(w http.ResponseWriter, r *http.Request) {
	tx, err := env.db.Begin()
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	err = r.ParseForm()
	if err != nil {
		respondWithErrorPage(w, http.StatusBadRequest, err)
		return
	}

	users, err := env.queries.GetUsers(tx)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	// This is here because the queries require a transaction even though this handler does not make any database changes.
	err = tx.Commit()
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	userOptions := []FormOption{}
	userSet := map[string]bool{}
	for _, user := range users {
		userSet[strconv.Itoa(user.Id)] = true
		userOptions = append(userOptions, FormOption{
			Id:   strconv.Itoa(user.Id),
			Name: user.Name,
		})
	}

	renderForm := func(userId string, idempotencyKey string, formErrors map[string]string) {
		data := struct {
			UserOptions    []FormOption
			UserId         string
			IdempotencyKey string
			FormErrors     map[string]string
		}{
			UserOptions:    userOptions,
			UserId:         userId,
			IdempotencyKey: idempotencyKey,
			FormErrors:     formErrors,
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

		err = ts.Execute(w, data)
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
	}

	if r.Method == http.MethodPost {
		formErrors := make(map[string]string)
		idempotencyKey := r.PostForm.Get("_idempotency_key")
		userId := r.PostForm.Get("user_id")
		password := r.PostForm.Get("password")

		if !userSet[userId] {
			formErrors["user_id"] = "Benutzer wählen"
		}

		if password != env.password {
			formErrors["password"] = "Passwort inkorrekt"
		}

		if len(formErrors) == 0 {
			intUserId, _ := strconv.Atoi(userId)
			env.session.Put(r, "user_id", intUserId)

			http.Redirect(w, r, "/plan", http.StatusSeeOther)
		} else {
			renderForm(userId, idempotencyKey, formErrors)
		}
	} else {
		renderForm("", IdempotencyKey(), make(map[string]string))
	}
}
