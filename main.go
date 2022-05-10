package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime/debug"
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

func (env *Environment) idempotencyKeysCleaner() {
	for {
		tx, err := env.db.Begin()
		if err != nil {
			panic(fmt.Sprintf("idempotencyKeysCleaner: %v", err))
		}

		_, err = env.queries.RemovePreviousIdempotencyKeys(tx)

		if err != nil {
			tx.Rollback()
			panic(fmt.Sprintf("idempotencyKeysCleaner: %v", err))
		} else {
			err = tx.Commit()
			if err != nil {
				panic(fmt.Sprintf("idempotencyKeysCleaner: %v", err))
			}
		}

		time.Sleep(60 * 60 * time.Second)
	}
}

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
	queries queries.Queries
	session *sessions.Session
	db      *sql.DB
}

type Configuration struct {
	Database string
	Secret   string
	Listen   string
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

	_, err = db.Exec("SELECT icu_load_collation('de_AT', 'de_AT');")
	if err != nil {
		log.Fatalln("Error loading collation: ", err.Error())
	}

	migrations.Run(db)

	session := sessions.New([]byte(configuration.Secret))
	session.Lifetime = 30 * 24 * time.Hour

	env := &Environment{
		queries: queries.Build(db),
		session: session,
		db:      db,
	}

	fileServer := http.FileServer(http.FS(web.Static))

	externalHandler := func(handler func(http.ResponseWriter, *http.Request)) http.Handler {
		return env.session.Enable(env.authenticate(http.HandlerFunc(handler)))
	}

	internalHandler := func(handler func(http.ResponseWriter, *http.Request)) http.Handler {
		return env.session.Enable(env.authenticate(env.requireAuthentication(http.HandlerFunc(handler))))
	}

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
			respondWithErrorPage(w, http.StatusBadRequest, err)
			return
		}

		id, err := types.UserIdFromString(r.PostForm.Get("user_id"))
		if err != nil {
			respondWithErrorPage(w, http.StatusBadRequest, err)
			return
		}

		tx, err := env.db.Begin()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		user, err := env.queries.GetUserById(tx, *id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				respondWithErrorPage(w, http.StatusNotFound, err)
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

		env.session.Put(r, "user_id", user.Id)

		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	} else {
		tx, err := env.db.Begin()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		users, err := env.queries.GetUsers(tx)
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
			"screens/identify.html",
			"layouts/external.html",
		}

		ts, err := template.ParseFS(web.Templates, files...)
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		err = ts.Execute(w, users)
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}
	}
}

func (env *Environment) HomeRoute(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct {
		CurrentUser types.User
	}{
		CurrentUser: user,
	}

	files := []string{
		"screens/home.html",
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

func (env *Environment) LogoutRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		env.session.Destroy(r)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		respondWithErrorPage(w, http.StatusBadRequest, errors.New("Logout muss per `POST` Methode passieren."))
	}
}

func IdempotencyKey() string {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return base64.RawURLEncoding.EncodeToString(key)[0:32]
}

func (env *Environment) PlanRoute(w http.ResponseWriter, r *http.Request) {
	tx, err := env.db.Begin()
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	addedItems, err := env.queries.GetAddedItems(tx)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	removedItems, err := env.queries.GetRemovedItems(tx)
	if err != nil {
		respondWithErrorPage(w, http.StatusInternalServerError, err)
		return
	}

	products, err := env.queries.GetProducts(tx)
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
		"screens/plan.html",
		"layouts/internal.html",
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct {
		CurrentUser    types.User
		Products       []types.Product
		AddedItems     []types.AddedItem
		RemovedItems   []types.AddedItem
		IdempotencyKey string
	}{
		CurrentUser:    user,
		Products:       products,
		AddedItems:     addedItems,
		RemovedItems:   removedItems,
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
			CategoryIds       map[string]bool
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

func (env *Environment) CheckItemRoute(w http.ResponseWriter, r *http.Request) {
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
		itemId, err := strconv.Atoi(r.PostForm.Get("item_id"))
		if err != nil {
			respondWithErrorPage(w, http.StatusBadRequest, err)
			return
		}

		successPath := fmt.Sprintf("/shop?sort-by=%s", sortBy)
		if sortBy == "" {
			successPath = "/shop"
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

		err = env.queries.SetItemState(tx, item.Id, "gathered")
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		err = env.queries.InsertItemChange(tx, int64(item.Id), user.Id, item.Dimension.Id, int64(item.Quantity), "gathered")
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		err = env.queries.InsertIdempotencyKey(tx, idempotencyKey)
		if err != nil {
			if err.Error() == "UNIQUE constraint failed: idempotency_keys.key" {
				http.Redirect(w, r, successPath, http.StatusSeeOther)
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

		http.Redirect(w, r, successPath, http.StatusSeeOther)
	} else {
		err = tx.Commit()
		if err != nil {
			respondWithErrorPage(w, http.StatusInternalServerError, err)
			return
		}

		http.Redirect(w, r, "/shop", http.StatusSeeOther)
	}
}

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
