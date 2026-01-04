package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (env *Environment) AddProductRoute(w http.ResponseWriter, r *http.Request) {
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

	categories, err := env.queries.GetCategories(tx)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	dimensions, err := env.queries.GetDimensions(tx)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	type CategoryOption struct {
		Id   string
		Name string
	}

	categoryOptions := []CategoryOption{}
	categorySet := map[string]bool{}
	for _, category := range categories {
		categorySet[strconv.Itoa(category.Id)] = true
		categoryOptions = append(categoryOptions, CategoryOption{
			Id:   strconv.Itoa(category.Id),
			Name: category.Name,
		})
	}

	dimensionOptions := []FormOption{}
	dimensionSet := map[string]bool{}
	for _, dimension := range dimensions {
		dimensionSet[strconv.Itoa(dimension.Id)] = true
		dimensionOptions = append(dimensionOptions, FormOption{
			Id:   strconv.Itoa(dimension.Id),
			Name: dimension.Name,
		})
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)

	renderForm := func(nameSingular string, namePlural string, categoryIds map[string]bool, dimensionIds map[string]bool, idempotencyKey string, formErrors map[string]string) {
		data := struct {
			CurrentUser      types.User
			Categories       []CategoryOption
			DimensionOptions []FormOption
			NameSingular     string
			NamePlural       string
			CategoryIds      map[string]bool
			DimensionIds     map[string]bool
			IdempotencyKey   string
			FormErrors       map[string]string
		}{
			CurrentUser:      user,
			Categories:       categoryOptions,
			NameSingular:     nameSingular,
			NamePlural:       namePlural,
			CategoryIds:      categoryIds,
			IdempotencyKey:   idempotencyKey,
			FormErrors:       formErrors,
			DimensionOptions: dimensionOptions,
			DimensionIds:     dimensionIds,
		}

		files := []string{
			"screens/add_product.html",
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

	if r.Method == http.MethodPost {
		formErrors := make(map[string]string)
		idempotencyKey := r.PostForm.Get("_idempotency_key")
		nameSingular := strings.TrimSpace(r.PostForm.Get("name_singular"))
		namePlural := strings.TrimSpace(r.PostForm.Get("name_plural"))
		categoryIds := r.PostForm["category_ids"]
		dimensionIds := r.PostForm["dimension_ids"]

		selectedDimensions := map[string]bool{}
		for _, id := range dimensionIds {
			selectedDimensions[id] = true
		}

		selectedCategories := map[string]bool{}
		for _, id := range categoryIds {
			selectedCategories[id] = true
		}

		if nameSingular == "" {
			formErrors["name_singular"] = "Namen angeben"
		} else if utf8.RuneCountInString(nameSingular) > 40 {
			formErrors["name_singular"] = "Kürzeren Namen angeben (maximal 40 Zeichen)"
		}

		if namePlural == "" {
			formErrors["name_plural"] = "Namen angeben"
		} else if utf8.RuneCountInString(namePlural) > 40 {
			formErrors["name_plural"] = "Kürzeren Namen angeben (maximal 40 Zeichen)"
		}

		if len(categoryIds) == 0 {
			formErrors["category_ids"] = "Kategorie wählen"
		}

		for _, categoryId := range categoryIds {
			if !categorySet[categoryId] {
				formErrors["category_ids"] = "Kategorie wählen"
			}
		}

		if len(dimensionIds) == 0 {
			formErrors["dimension_ids"] = "Größenordnung wählen"
		}

		for _, dimensionId := range dimensionIds {
			if !dimensionSet[dimensionId] {
				formErrors["dimension_ids"] = "Größenordnung wählen"
			}
		}

		if len(formErrors) == 0 {
			result, err := env.queries.InsertProduct(tx, nameSingular, namePlural)
			if err != nil {
				if err.Error() == "UNIQUE constraint failed: index 'idx_products_name_singular'" {
					formErrors["name_singular"] = "Anderen Namen angeben (ist bereits in Verwendung)"
					renderForm(nameSingular, namePlural, selectedCategories, selectedDimensions, idempotencyKey, formErrors)
					return
				} else if err.Error() == "UNIQUE constraint failed: index 'idx_products_name_plural'" {
					formErrors["name_plural"] = "Anderen Namen angeben (ist bereits in Verwendung)"
					renderForm(nameSingular, namePlural, selectedCategories, selectedDimensions, idempotencyKey, formErrors)
					return
				} else {
					respondWithErrorPage(w, http.StatusInternalServerError, err)
					return
				}
			}
			productId, err := result.LastInsertId()
			if err != nil {
				respondWithErrorPage(w, http.StatusInternalServerError, err)
				return
			}

			for _, id := range dimensionIds {
				result, err = env.queries.InsertProductDimension(tx, productId, id)
				if err != nil {
					respondWithErrorPage(w, http.StatusInternalServerError, err)
					return
				}
			}

			for _, id := range categoryIds {
				result, err = env.queries.InsertProductCategory(tx, productId, id)
				if err != nil {
					respondWithErrorPage(w, http.StatusInternalServerError, err)
					return
				}
			}

			_, err = env.queries.InsertProductChange(tx, productId, user.Id, nameSingular, namePlural)
			if err != nil {
				respondWithErrorPage(w, http.StatusInternalServerError, err)
				return
			}

			err = env.queries.InsertIdempotencyKey(tx, idempotencyKey)
			if err != nil {
				if err.Error() == "UNIQUE constraint failed: idempotency_keys.key" {
					http.Redirect(w, r, "/plan", http.StatusSeeOther)
					return
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

			http.Redirect(w, r, fmt.Sprintf("/add-item?product-id=%d", productId), http.StatusSeeOther)
		} else {
			err = tx.Commit()
			if err != nil {
				respondWithErrorPage(w, http.StatusInternalServerError, err)
				return
			}

			renderForm(nameSingular, namePlural, selectedCategories, selectedDimensions, idempotencyKey, formErrors)
		}
	} else {
		name := r.Form.Get("name")
		product, err := env.queries.GetProductByName(tx, name)

		err2 := tx.Commit()
		if err2 != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err2)
			return
		}

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				renderForm(name, name, make(map[string]bool), make(map[string]bool), IdempotencyKey(), make(map[string]string))
			} else {
				respondWithErrorPage(w, http.StatusInternalServerError, err)
				return
			}
		} else {
			http.Redirect(w, r, fmt.Sprintf("/add-item?product-id=%d", product.Id), http.StatusSeeOther)
		}
	}
}
