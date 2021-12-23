package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golangcollege/sessions"
	_ "github.com/mattn/go-sqlite3"
	"stravid.com/besserliste/migrations"
	"stravid.com/besserliste/queries"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
)

type contextKey string
const contextKeyCurrentUser = contextKey("currentUser")

func (env *Environment) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exists := env.session.Exists(r, "user_id")
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		user, err := env.queries.GetUserById(env.session.GetInt(r, "user_id"))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				env.session.Remove(r, "user_id")
				next.ServeHTTP(w, r)
				return
			} else {
				log.Println(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
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
	queries queries.Queries
	session *sessions.Session
}

type Configuration struct {
	Database          string
	Secret          string
}

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

	migrations.Run(db)

	session := sessions.New([]byte(configuration.Secret))
	session.Lifetime = 48 * time.Hour

	rand.Seed(time.Now().UnixNano())

	env := &Environment{
		queries: queries.Build(db),
		session: session,
	}

	fileServer := http.FileServer(http.FS(web.Static))

	externalHandler := func(handler func(http.ResponseWriter, *http.Request)) http.Handler {
		return env.session.Enable(env.authenticate(http.HandlerFunc(handler)))
	}

	internalHandler := func(handler func(http.ResponseWriter, *http.Request)) http.Handler {
		return env.session.Enable(env.authenticate(env.requireAuthentication(http.HandlerFunc(handler))))
	}

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

	err = http.ListenAndServe(":4000", mux)
	if err != nil {
		log.Fatalln("Error starting Besserliste web application: ", err.Error())
	}
}

func (env *Environment) RootRoute(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (env *Environment) LoginRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		id, err := types.UserIdFromString(r.PostForm.Get("user_id"))
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		user, err := env.queries.GetUserById(*id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "User not found", http.StatusNotFound)
			} else {
				log.Println(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}

		env.session.Put(r, "user_id", user.Id)

		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	} else {
		users, err := env.queries.GetUsers()
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		files := []string{
			"screens/identify.html",
			"layouts/external.html",
		}

		ts, err := template.ParseFS(web.Templates, files...)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = ts.Execute(w, users)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

func (env *Environment) LogoutRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		env.session.Destroy(r)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}


func (env *Environment) PlanRoute(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"screens/plan.html",
		"layouts/internal.html",
	}

	products := []string{}

	for i := 0; i < 1000; i++ {
	  products = append(products, randSeq(10))
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct{
		CurrentUser types.User
		Products []string
	} {
		CurrentUser: user,
		Products: products,
	}

	ts, err := template.ParseFS(web.Templates, files...)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = ts.Execute(w, data)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (env *Environment) AddProductRoute(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	categories, err := env.queries.GetCategories()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	type CategoryOption struct {
		Id string
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

	if r.Method == http.MethodPost {
		errors := make(map[string]string)
		name := strings.TrimSpace(r.PostForm.Get("name"))
		categoryId := r.PostForm.Get("category_id")

		if name == "" {
			errors["name"] = "Name eingeben"
		} else if utf8.RuneCountInString(name) > 40 {
			errors["name"] = "Kürzeren Namen eingeben (maximal 40 Zeichen)"
		}

		if !categorySet[categoryId] {
			errors["category_id"] = "Kategorie wählen"
		}

		if len(errors) > 0 {
			files := []string{
				"screens/add_product.html",
				"layouts/internal.html",
			}

			user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
			data := struct{
				CurrentUser types.User
				Categories []CategoryOption
				Name string
				CategoryId string
				FormErrors map[string]string
			} {
				CurrentUser: user,
				Categories: categoryOptions,
				Name: name,
				CategoryId: categoryId,
				FormErrors: errors,
			}

			ts, err := template.ParseFS(web.Templates, files...)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			err = ts.Execute(w, data)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		} else {
			http.Redirect(w, r, "/plan", http.StatusSeeOther)
		}
	} else {
		name := r.Form.Get("name")

		files := []string{
			"screens/add_product.html",
			"layouts/internal.html",
		}

		user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
		data := struct{
			CurrentUser types.User
			Categories []CategoryOption
			Name string
			CategoryId string
			FormErrors map[string]string
		} {
			CurrentUser: user,
			Categories: categoryOptions,
			Name: name,
			CategoryId: "",
		}

		ts, err := template.ParseFS(web.Templates, files...)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = ts.Execute(w, data)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

func (env *Environment) AddItemRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	} else {
		files := []string{
			"screens/add_item.html",
			"layouts/internal.html",
		}

		user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
		data := struct{
			CurrentUser types.User
		} {
			CurrentUser: user,
		}

		ts, err := template.ParseFS(web.Templates, files...)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = ts.Execute(w, data)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}

func (env *Environment) ShopRoute(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"screens/shop.html",
		"layouts/internal.html",
	}

	ts, err := template.ParseFS(web.Templates, files...)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct{
		CurrentUser types.User
	} {
		CurrentUser: user,
	}

	err = ts.Execute(w, data)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (env *Environment) CheckItemRoute(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/shop", http.StatusSeeOther)
}
