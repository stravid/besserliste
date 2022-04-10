package main

import (
	"log"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	"stravid.com/besserliste/types"
)

func (env *Environment) UndoRoute(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)

	if r.Method == http.MethodPost {
		idempotencyKey := r.PostForm.Get("_idempotency_key")
		oldState := r.PostForm.Get("old_state")
		newState := r.PostForm.Get("new_state")
		itemId, err := strconv.Atoi(r.PostForm.Get("item_id"))
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		item, err := env.queries.GetItem(itemId)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if item.State != oldState {
			log.Printf("Expected state %s but got %s", oldState, item.State)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		tx, err := env.db.Begin()
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec("UPDATE items SET state = ? WHERE id = ?", newState, item.Id)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT INTO item_changes (item_id, user_id, dimension_id, quantity, state, recorded_at) VALUES (?, ?, ?, ?, ?, datetime('now'))", item.Id, user.Id, item.Dimension.Id, item.Quantity, newState)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT INTO idempotency_keys (key, processed_at) VALUES (?, datetime('now'))", idempotencyKey)
		if err != nil {
			if err.Error() == "UNIQUE constraint failed: idempotency_keys.key" {
				http.Redirect(w, r, "/plan", http.StatusSeeOther)
				return
			} else {
				log.Println(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		err = tx.Commit()
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if oldState == "removed" {
			http.Redirect(w, r, "/plan", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/shop", http.StatusSeeOther)
		}
	} else {
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}
