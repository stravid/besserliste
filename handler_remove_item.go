package main

import (
	"errors"
	"net/http"
	"stravid.com/besserliste/types"
	"strconv"
)

func (env *Environment) RemoveItemRoute(w http.ResponseWriter, r *http.Request) {
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

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)

	if r.Method == http.MethodPost {
		idempotencyKey := r.PostForm.Get("_idempotency_key")
		itemId, err := strconv.Atoi(r.PostForm.Get("item_id"))
		if err != nil {
			respondWithErrorPage(w, http.StatusBadRequest, err)
			return
		}

		item, err := env.queries.GetItem(tx, itemId)
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		if item.State != "added" {
			respondWithErrorPage(w, http.StatusBadRequest, errors.New("Eintrag befindet sich im falschen Zustand."))
			return
		}

		err = env.queries.SetItemState(tx, item.Id, "removed")
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		err = env.queries.InsertItemChange(tx, int64(item.Id), user.Id, item.Dimension.Id, int64(item.Quantity), "removed")
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
		err = tx.Commit()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	}
}
