package main

import (
	"log"
	"net/http"
	"html/template"
	"stravid.com/besserliste/web"
	"time"
    "math/rand"
)

type Environment struct {}

func main() {
	rand.Seed(time.Now().UnixNano())
	env := &Environment{}
	fileServer := http.FileServer(http.FS(web.Static))

	mux := http.NewServeMux()
	mux.Handle("/staticss/", fileServer)
	mux.Handle("/", http.HandlerFunc(env.RootRoute))
	mux.Handle("/login", http.HandlerFunc(env.LoginRoute))
	mux.Handle("/plan", http.HandlerFunc(env.PlanRoute))
	mux.Handle("/add-product", http.HandlerFunc(env.AddProductRoute))
	mux.Handle("/add-item", http.HandlerFunc(env.AddItemRoute))
	mux.Handle("/shop", http.HandlerFunc(env.ShopRoute))
	mux.Handle("/check-item", http.HandlerFunc(env.CheckItemRoute))

	err := http.ListenAndServe(":4000", mux)
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
		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	} else {
		files := []string{
			"screens/identify.html",
		}

		ts, err := template.ParseFS(web.Templates, files...)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = ts.Execute(w, "")
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
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
	}

	products := []string{}

	for i := 0; i < 1000; i++ {
	  products = append(products, randSeq(10))
	}

	ts, err := template.ParseFS(web.Templates, files...)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = ts.Execute(w, products)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (env *Environment) AddProductRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	} else {
		files := []string{
			"screens/add_product.html",
		}

		ts, err := template.ParseFS(web.Templates, files...)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = ts.Execute(w, "")
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
		}

		ts, err := template.ParseFS(web.Templates, files...)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = ts.Execute(w, "")
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
	}

	ts, err := template.ParseFS(web.Templates, files...)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = ts.Execute(w, "")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (env *Environment) CheckItemRoute(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/shop", http.StatusSeeOther)
}
