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
)

func (env *Environment) AddItemRoute(w http.ResponseWriter, r *http.Request) {
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

	productId, err := strconv.Atoi(r.Form.Get("product-id"))
	if err != nil {
		respondWithErrorPage(w, http.StatusBadRequest, err)
		return
	}

	product, err := env.queries.GetProduct(tx, productId)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	units := []types.Unit{}
	for _, dimension := range product.Dimensions {
		for _, unit := range dimension.Units {
			units = append(units, unit)
		}
	}

	unitOptions := []FormOption{}
	unitSet := map[string]bool{}
	for _, unit := range units {
		unitSet[strconv.Itoa(unit.Id)] = true
		unitOptions = append(unitOptions, FormOption{
			Id:   strconv.Itoa(unit.Id),
			Name: unit.NamePlural,
		})
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)

	renderForm := func(quantity string, unitId string, idempotencyKey string, formErrors map[string]string) {
		files := []string{
			"screens/add_item.html",
			"layouts/internal.html",
		}

		data := struct {
			CurrentUser    types.User
			Product        types.SelectedProduct
			UnitOptions    []FormOption
			IdempotencyKey string
			FormErrors     map[string]string
			Quantity       string
			UnitId         string
		}{
			CurrentUser:    user,
			Product:        *product,
			UnitOptions:    unitOptions,
			Quantity:       quantity,
			UnitId:         unitId,
			IdempotencyKey: idempotencyKey,
			FormErrors:     formErrors,
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
		unitId := r.PostForm.Get("unit_id")
		amount := r.PostForm.Get("quantity")
		parsedQuantity, amountErr := strconv.ParseFloat(strings.Replace(amount, ",", ".", -1), 64)

		if !unitSet[unitId] {
			formErrors["unit_id"] = "Maßeinheit wählen"
		}

		if amount == "" {
			formErrors["quantity"] = "Menge angeben"
		} else if amountErr != nil {
			formErrors["quantity"] = "Zahl angeben"
		}

		if len(formErrors) == 0 {
			unit := types.Unit{}
			dimension := types.Dimension{}

			for _, u := range units {
				if i, _ := strconv.Atoi(unitId); i == u.Id {
					unit = u
					break
				}
			}

			for _, d := range product.Dimensions {
				for _, u := range d.Units {
					if u == unit {
						dimension = d
						break
					}
				}
			}

			startQuantiy := int64(0)
			itemId := int64(0)
			item, err := env.queries.GetAddedItemByProductDimension(tx, product.Id, dimension.Id)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					respondWithErrorPage(w, http.StatusInternalServerError, err)
					return
				}
			} else {
				startQuantiy = int64(item.Quantity)
				itemId = int64(item.Id)
			}

			remainingQuanity := int64(10000 - startQuantiy)
			parsedBaseQuantity := parsedQuantity * unit.ConversionToBase
			baseQuantity := int64(parsedBaseQuantity)

			if parsedBaseQuantity != float64(baseQuantity) {
				formErrors["quantity"] = "Ganze Zahl angeben"
			}

			if baseQuantity < 1 {
				formErrors["amount"] = "Größere Menge angeben (kleinste Menge ist 1)"
			}

			if baseQuantity > remainingQuanity {
				formErrors["amount"] = fmt.Sprintf("Kleinere Menge angeben (größte Menge ist %d)", int64(unit.ConversionFromBase*float64(remainingQuanity)))
			}

			if len(formErrors) == 0 {
				if itemId == 0 {
					result, err := env.queries.InsertItem(tx, product.Id, dimension.Id, baseQuantity+startQuantiy)
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}

					itemId, err = result.LastInsertId()
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}
				} else {
					err = env.queries.SetItemQuantity(tx, int64(item.Id), baseQuantity+startQuantiy)
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}
				}

				err = env.queries.InsertItemChange(tx, itemId, user.Id, dimension.Id, baseQuantity+startQuantiy, "added")
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

				http.Redirect(w, r, "/plan", http.StatusSeeOther)
			} else {
				renderForm(amount, unitId, idempotencyKey, formErrors)
			}
		} else {
			renderForm(amount, unitId, idempotencyKey, formErrors)
		}
	} else {
		err = tx.Commit()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		renderForm("", "", IdempotencyKey(), make(map[string]string))
	}
}
