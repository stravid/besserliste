package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
	"strconv"
)

func (env *Environment) ShopRoute(w http.ResponseWriter, r *http.Request) {
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

	sortBy := r.Form.Get("sort-by")
	sortSet := map[string]bool{
		"": true,
	}
	sortOptions := []FormOption{
		{Id: "", Name: "Alphabetisch"},
	}

	categories, err := env.queries.GetCategories(tx)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	for _, category := range categories {
		sortOptions = append(sortOptions, FormOption{
			Id:   strconv.Itoa(category.Id),
			Name: category.Name,
		})
		sortSet[strconv.Itoa(category.Id)] = true
	}

	if !sortSet[sortBy] {
		respondWithErrorPage(w, http.StatusBadRequest, errors.New(fmt.Sprintf("Unbekannter Wert `%s` für `sort-by`.", sortBy)))
		return
	}

	var addedItems []types.AddedItem
	if sortBy == "" {
		addedItems, err = env.queries.GetRemainingItemsByAlphabet(tx)
	} else {
		categoryId, err := strconv.Atoi(sortBy)

		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		addedItems, err = env.queries.GetRemainingItemsByCategory(tx, categoryId)
	}

	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	gatheredItems, err := env.queries.GetGatheredItems(tx)
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
		"screens/shop.html",
		"layouts/internal.html",
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct {
		CurrentUser    types.User
		Products       []types.Product
		AddedItems     []types.AddedItem
		GatheredItems  []types.AddedItem
		SortOptions    []FormOption
		SortBy         string
		IdempotencyKey string
	}{
		CurrentUser:    user,
		AddedItems:     addedItems,
		GatheredItems:  gatheredItems,
		SortOptions:    sortOptions,
		SortBy:         sortBy,
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
