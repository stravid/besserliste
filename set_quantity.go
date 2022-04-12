package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
)

func (env *Environment) SetQuantityRoute(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		respondWithErrorPage(w, http.StatusBadRequest, err)
		return
	}

	itemId, err := strconv.Atoi(r.Form.Get("item-id"))
	if err != nil {
		respondWithErrorPage(w, http.StatusBadRequest, err)
		return
	}

	item, err := env.queries.GetItem(itemId)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	product, err := env.queries.GetProduct(item.ProductId)

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
			"screens/set_quantity.html",
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
			itemIdForSelectedDimension := int64(0)
			itemForSelectedDimension, err := env.queries.GetAddedItemByProductDimension(product.Id, dimension.Id)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					respondWithErrorPage(w, http.StatusInternalServerError, err)
					return
				}
			} else {
				startQuantiy = int64(itemForSelectedDimension.Quantity)
				itemIdForSelectedDimension = int64(itemForSelectedDimension.Id)
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
				tx, err := env.db.Begin()
				if err != nil {
					respondWithErrorPage(w, http.StatusInternalServerError, err)
					return
				}
				defer tx.Rollback()

				initialItemNeedsToBeRemoved := item.Dimension.Id != dimension.Id && itemIdForSelectedDimension != 0

				if initialItemNeedsToBeRemoved {
					// Remove selected item
					_, err := tx.Exec("UPDATE items SET state = 'removed' WHERE id = ?", item.Id)
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}

					_, err = tx.Exec("INSERT INTO item_changes (item_id, user_id, dimension_id, quantity, state, recorded_at) VALUES (?, ?, ?, ?, 'removed', datetime('now'))", itemId, user.Id, item.Dimension.Id, item.Quantity)
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}

					// Update existing item
					_, err = tx.Exec("UPDATE items SET quantity = ? WHERE id = ?", baseQuantity+startQuantiy, itemIdForSelectedDimension)
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}

					_, err = tx.Exec("INSERT INTO item_changes (item_id, user_id, dimension_id, quantity, state, recorded_at) VALUES (?, ?, ?, ?, 'added', datetime('now'))", itemIdForSelectedDimension, user.Id, itemForSelectedDimension.Dimension.Id, baseQuantity+startQuantiy)
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}
				} else {
					// Update existing item
					_, err := tx.Exec("UPDATE items SET quantity = ?, dimension_id = ? WHERE id = ?", baseQuantity, dimension.Id, itemId)
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}

					_, err = tx.Exec("INSERT INTO item_changes (item_id, user_id, dimension_id, quantity, state, recorded_at) VALUES (?, ?, ?, ?, 'added', datetime('now'))", itemId, user.Id, dimension.Id, baseQuantity)
					if err != nil {
						respondWithErrorPage(w, http.StatusInternalServerError, err)
						return
					}
				}

				_, err = tx.Exec("INSERT INTO idempotency_keys (key, processed_at) VALUES (?, datetime('now'))", idempotencyKey)
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
		floatQuantity := float64(item.Quantity)
		preselectedUnit := types.Unit{}
		for _, u := range item.Dimension.Units {
			if preselectedUnit.Id == 0 {
				preselectedUnit = u
			} else {
				newUnitQuantityIsSmaller := preselectedUnit.ConversionFromBase*floatQuantity > u.ConversionFromBase*floatQuantity
				newUnitQuantityIsBigEnough := u.ConversionFromBase*floatQuantity >= 1

				if newUnitQuantityIsSmaller && newUnitQuantityIsBigEnough {
					preselectedUnit = u
				}
			}
		}

		formattedQuantity := strings.Replace(strconv.FormatFloat(preselectedUnit.ConversionFromBase*floatQuantity, 'f', -1, 32), ".", ",", -1)

		renderForm(formattedQuantity, strconv.Itoa(preselectedUnit.Id), IdempotencyKey(), make(map[string]string))
	}
}
