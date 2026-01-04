package main

import (
	"html/template"
	"net/http"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
)

func (env *Environment) PlanRoute(w http.ResponseWriter, r *http.Request) {
	tx, err := env.db.Begin()
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	addedItems, err := env.queries.GetAddedItems(tx)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	removedItems, err := env.queries.GetRemovedItems(tx)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	products, err := env.queries.GetProducts(tx)
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
		"screens/plan.html",
		"layouts/internal.html",
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct {
		CurrentUser    types.User
		Products       []types.Product
		AddedItems     []types.AddedItem
		RemovedItems   []types.AddedItem
		IdempotencyKey string
	}{
		CurrentUser:    user,
		Products:       products,
		AddedItems:     addedItems,
		RemovedItems:   removedItems,
		IdempotencyKey: IdempotencyKey(),
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
