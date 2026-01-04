package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golangcollege/sessions"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"stravid.com/besserliste/migrations"
	"stravid.com/besserliste/queries"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
	"time"
)

func main() {
	configurationFile, _ := os.Open("config.json")
	defer configurationFile.Close()
	decoder := json.NewDecoder(configurationFile)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Fatalln("Error opening config.json: ", err.Error())
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_foreign_keys=on", configuration.Database))
	if err != nil {
		log.Fatalln("Error opening database: ", err.Error())
	}
	defer db.Close()

	// Load correct collation so our sorting works as expected.
	_, err = db.Exec("SELECT icu_load_collation('de_AT', 'de_AT');")
	if err != nil {
		log.Fatalln("Error loading collation: ", err.Error())
	}

	// Run migrations at boot to get current database schema.
	migrations.Run(db)

	session := sessions.New([]byte(configuration.Secret))
	session.Lifetime = 30 * 24 * time.Hour

	env := &Environment{
		queries:  queries.Build(db),
		session:  session,
		db:       db,
		password: configuration.Password,
	}

	fileServer := http.FileServer(http.FS(web.Static))

	// External HTTP handlers tolerate anonymous users.
	externalHandler := func(handler func(http.ResponseWriter, *http.Request)) http.Handler {
		return env.session.Enable(env.authenticate(http.HandlerFunc(handler)))
	}

	// Internal HTTP handlers require a signed-in user.
	internalHandler := func(handler func(http.ResponseWriter, *http.Request)) http.Handler {
		return env.session.Enable(env.authenticate(env.requireAuthentication(http.HandlerFunc(handler))))
	}

	// Start background Go routine that periodically removes old idempotency keys.
	go env.idempotencyKeysCleaner()

	mux := http.NewServeMux()
	mux.Handle("/static/", fileServer)
	mux.Handle("/", internalHandler(env.RootRoute))
	mux.Handle("/login", externalHandler(env.LoginRoute))
	mux.Handle("/logout", internalHandler(env.LogoutRoute))
	mux.Handle("/plan", internalHandler(env.PlanRoute))
	mux.Handle("/add-product", internalHandler(env.AddProductRoute))
	mux.Handle("/add-item", internalHandler(env.AddItemRoute))
	mux.Handle("/shop", internalHandler(env.ShopRoute))
	mux.Handle("/check-item", internalHandler(env.CheckItemRoute))
	mux.Handle("/remove-item", internalHandler(env.RemoveItemRoute))
	mux.Handle("/home", internalHandler(env.HomeRoute))
	mux.Handle("/undo", internalHandler(env.UndoRoute))
	mux.Handle("/set-quantity", internalHandler(env.SetQuantityRoute))

	err = http.ListenAndServe(configuration.Listen, mux)
	if err != nil {
		log.Fatalln("Error starting Besserliste web application: ", err.Error())
	}
}

func respondWithErrorPage(w http.ResponseWriter, statusCode int, err error) {
	log.Println(fmt.Sprintf("%s\n%s", err.Error(), debug.Stack()))

	data := struct {
		Error   string
		Message string
	}{
		Error:   http.StatusText(statusCode),
		Message: err.Error(),
	}

	files := []string{
		"layouts/error.html",
	}

	ts, err := template.ParseFS(web.Templates, files...)
	if err != nil {
		log.Println(err.Error())
		panic("Error in error page logic.")
	}

	w.WriteHeader(statusCode)
	err = ts.Execute(w, data)
	if err != nil {
		log.Println(err.Error())
		panic("Error in error page logic.")
	}
}

type FormOption struct {
	Id   string
	Name string
}

type contextKey string

const contextKeyCurrentUser = contextKey("currentUser")

func (env *Environment) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exists := env.session.Exists(r, "user_id")
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		tx, err := env.db.Begin()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		user, err := env.queries.GetUserById(tx, env.session.GetInt(r, "user_id"))

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				env.session.Remove(r, "user_id")
				next.ServeHTTP(w, r)
				return
			} else {
				respondWithErrorPage(w, http.StatusInternalServerError, err)
			}
		}

		err = tx.Commit()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyCurrentUser, *user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (env *Environment) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !env.isAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (env *Environment) isAuthenticated(r *http.Request) bool {
	_, ok := r.Context().Value(contextKeyCurrentUser).(types.User)

	if !ok {
		return false
	} else {
		return true
	}
}

type Environment struct {
	queries  queries.Queries
	session  *sessions.Session
	db       *sql.DB
	password string
}

type Configuration struct {
	Database string
	Secret   string
	Listen   string
	Password string
}
