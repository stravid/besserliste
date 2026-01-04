package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	"stravid.com/besserliste/types"
)

func (env *Environment) UndoRoute(w http.ResponseWriter, r *http.Request) {
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
		sortBy := r.PostForm.Get("sort_by")
		oldState := r.PostForm.Get("old_state")
		newState := r.PostForm.Get("new_state")
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

		if item.State != oldState {
			respondWithErrorPage(w, http.StatusInternalServerError, errors.New(fmt.Sprintf("Expected state %s but got %s", oldState, item.State)))
			return
		}

		err = env.queries.SetItemState(tx, item.Id, newState)
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		err = env.queries.InsertItemChange(tx, int64(item.Id), user.Id, item.Dimension.Id, int64(item.Quantity), newState)
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

		if oldState == "removed" {
			http.Redirect(w, r, "/plan", http.StatusSeeOther)
		} else {
			successPath := fmt.Sprintf("/shop?sort-by=%s", sortBy)
			if sortBy == "" {
				successPath = "/shop"
			}
			http.Redirect(w, r, successPath, http.StatusSeeOther)
		}
	} else {
		err = tx.Commit()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
	}
}
