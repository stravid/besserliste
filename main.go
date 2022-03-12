package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"crypto/rand"
	"encoding/base64"

	"github.com/golangcollege/sessions"
	_ "github.com/mattn/go-sqlite3"
	"stravid.com/besserliste/migrations"
	"stravid.com/besserliste/queries"
	"stravid.com/besserliste/types"
	"stravid.com/besserliste/web"
)

type FormOption struct {
	Id string
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
	db *sql.DB
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

	env := &Environment{
		queries: queries.Build(db),
		session: session,
		db: db,
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
	mux.Handle("/remove-item", internalHandler(env.RemoveItemRoute))

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

func IdempotencyKey() string {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return base64.RawURLEncoding.EncodeToString(key)[0:32]
}

func (env *Environment) PlanRoute(w http.ResponseWriter, r *http.Request) {
	addedItems, err := env.queries.GetAddedItems()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	products, err := env.queries.GetProducts()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	files := []string{
		"screens/plan.html",
		"layouts/internal.html",
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct{
		CurrentUser types.User
		Products []types.Product
		AddedItems []types.AddedItem
		IdempotencyKey string
	} {
		CurrentUser: user,
		Products: products,
		AddedItems: addedItems,
		IdempotencyKey: IdempotencyKey(),
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

	dimensions, err := env.queries.GetDimensions()
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

	dimensionOptions := []FormOption{}
	dimensionSet := map[string]bool{}
	for _, dimension := range dimensions {
		dimensionSet[strconv.Itoa(dimension.Id)] = true
		dimensionOptions = append(dimensionOptions, FormOption{
			Id: strconv.Itoa(dimension.Id),
			Name: dimension.Name,
		})
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)

	renderForm := func(nameSingular string, namePlural string, categoryId string, dimensionIds map[string]bool, idempotencyKey string, formErrors map[string]string) {
		data := struct{
				CurrentUser types.User
				Categories []CategoryOption
				DimensionOptions []FormOption
				NameSingular string
				NamePlural string
				CategoryId string
				DimensionIds map[string]bool
				IdempotencyKey string
				FormErrors map[string]string
			} {
				CurrentUser: user,
				Categories: categoryOptions,
				NameSingular: nameSingular,
				NamePlural: namePlural,
				CategoryId: categoryId,
				IdempotencyKey: idempotencyKey,
				FormErrors: formErrors,
				DimensionOptions: dimensionOptions,
				DimensionIds: dimensionIds,
			}

		files := []string{
			"screens/add_product.html",
			"layouts/internal.html",
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

	if r.Method == http.MethodPost {
		formErrors := make(map[string]string)
		idempotencyKey := r.PostForm.Get("_idempotency_key")
		nameSingular := strings.TrimSpace(r.PostForm.Get("name_singular"))
		namePlural := strings.TrimSpace(r.PostForm.Get("name_plural"))
		categoryId := r.PostForm.Get("category_id")
		dimensionIds := r.PostForm["dimension_ids"]

		selectedDimensions := map[string]bool{}
		for _, id := range dimensionIds {
			selectedDimensions[id] = true
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

		if !categorySet[categoryId] {
			formErrors["category_id"] = "Kategorie wählen"
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
			tx, err := env.db.Begin()
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			defer tx.Rollback()

			result, err := tx.Exec("INSERT INTO products (category_id, name_singular, name_plural) VALUES (?, ?, ?)", categoryId, nameSingular, namePlural)
			if err != nil {
				if err.Error() == "UNIQUE constraint failed: products.name_singular" {
					formErrors["name_singular"] = "Anderen Namen angeben (ist bereits in Verwendung)"
					renderForm(nameSingular, namePlural, categoryId, selectedDimensions, idempotencyKey, formErrors)
					return
				} else if err.Error() == "UNIQUE constraint failed: products.name_plural" {
					formErrors["name_plural"] = "Anderen Namen angeben (ist bereits in Verwendung)"
					renderForm(nameSingular, namePlural, categoryId, selectedDimensions, idempotencyKey, formErrors)
					return
				} else {
					log.Println(err.Error())
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}
			productId, err := result.LastInsertId()
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			for _, id := range dimensionIds {
				result, err = tx.Exec("INSERT INTO dimensions_products (dimension_id, product_id) VALUES (?, ?)", id, productId)
				if err != nil {
					log.Println(err.Error())
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}

			_, err = tx.Exec("INSERT INTO product_changes (product_id, user_id, category_id, name_singular, name_plural, recorded_at) VALUES (?, ?, ?, ?, ?, datetime('now'))", productId, user.Id, categoryId, nameSingular, namePlural)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			_, err = tx.Exec("INSERT INTO idempotency_keys (key, processed_at) VALUES (?, datetime('now'))", idempotencyKey)
			if err != nil {
				if (err.Error() == "UNIQUE constraint failed: idempotency_keys.key") {
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

			http.Redirect(w, r, fmt.Sprintf("/add-item?product-id=%d", productId), http.StatusSeeOther)
		} else {
			renderForm(nameSingular, namePlural, categoryId, selectedDimensions, idempotencyKey, formErrors)
		}
	} else {
		renderForm(r.Form.Get("name"), r.Form.Get("name"), "", make(map[string]bool), IdempotencyKey(), make(map[string]string))
	}
}

func (env *Environment) AddItemRoute(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	productId, err := strconv.Atoi(r.Form.Get("product-id"))
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	product, err := env.queries.GetProduct(productId)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
			Id: strconv.Itoa(unit.Id),
			Name: unit.NamePlural,
		})
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)

	renderForm := func(quantity string, unitId string, idempotencyKey string, formErrors map[string]string) {
		files := []string{
			"screens/add_item.html",
			"layouts/internal.html",
		}

		data := struct{
			CurrentUser types.User
			Product types.SelectedProduct
			UnitOptions []FormOption
			IdempotencyKey string
			FormErrors map[string]string
			Quantity string
			UnitId string
		} {
			CurrentUser: user,
			Product: *product,
			UnitOptions: unitOptions,
			Quantity: quantity,
			UnitId: unitId,
			IdempotencyKey: idempotencyKey,
			FormErrors: formErrors,
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
			item, err := env.queries.GetAddedItemByProductDimension(product.Id, dimension.Id)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					log.Println(err.Error())
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
				formErrors["amount"] = fmt.Sprintf("Kleinere Menge angeben (größte Menge ist %d)", int64(unit.ConversionFromBase * float64(remainingQuanity)))
			}

			if len(formErrors) == 0 {
				tx, err := env.db.Begin()
				if err != nil {
					log.Println(err.Error())
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				defer tx.Rollback()

				if itemId == 0 {
					result, err := tx.Exec("INSERT INTO items (product_id, dimension_id, quantity, state) VALUES (?, ?, ?, 'added')", product.Id, dimension.Id, baseQuantity + startQuantiy)
					if err != nil {
						log.Println(err.Error())
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}

					itemId, err = result.LastInsertId()
					if err != nil {
						log.Println(err.Error())
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
				} else {
					_, err := tx.Exec("UPDATE items SET quantity = ? WHERE id = ?", baseQuantity + startQuantiy, item.Id)
					if err != nil {
						log.Println(err.Error())
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
				}

				_, err = tx.Exec("INSERT INTO item_changes (item_id, user_id, quantity, state, recorded_at) VALUES (?, ?, ?, 'added', datetime('now'))", itemId, user.Id, baseQuantity + startQuantiy)
				if err != nil {
					log.Println(err.Error())
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				_, err = tx.Exec("INSERT INTO idempotency_keys (key, processed_at) VALUES (?, datetime('now'))", idempotencyKey)
				if err != nil {
					if (err.Error() == "UNIQUE constraint failed: idempotency_keys.key") {
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

				http.Redirect(w, r, "/plan", http.StatusSeeOther)
			} else {
				renderForm(amount, unitId, idempotencyKey, formErrors)
			}
		} else {
			renderForm(amount, unitId, idempotencyKey, formErrors)
		}
	} else {
		renderForm("", "", IdempotencyKey(), make(map[string]string))
	}
}

func (env *Environment) ShopRoute(w http.ResponseWriter, r *http.Request) {
	addedItems, err := env.queries.GetRemainingItemsByAlphabet()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	files := []string{
		"screens/shop.html",
		"layouts/internal.html",
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)
	data := struct{
		CurrentUser types.User
		Products []types.Product
		AddedItems []types.AddedItem
		IdempotencyKey string
	} {
		CurrentUser: user,
		AddedItems: addedItems,
		IdempotencyKey: IdempotencyKey(),
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

func (env *Environment) CheckItemRoute(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)

	if r.Method == http.MethodPost {
		idempotencyKey := r.PostForm.Get("_idempotency_key")
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

		tx, err := env.db.Begin()
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec("UPDATE items SET state = 'gathered' WHERE id = ?", item.Id)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT INTO item_changes (item_id, user_id, quantity, state, recorded_at) VALUES (?, ?, ?, 'gathered', datetime('now'))", item.Id, user.Id, item.Quantity)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT INTO idempotency_keys (key, processed_at) VALUES (?, datetime('now'))", idempotencyKey)
		if err != nil {
			if (err.Error() == "UNIQUE constraint failed: idempotency_keys.key") {
				http.Redirect(w, r, "/shop", http.StatusSeeOther)
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

		http.Redirect(w, r, "/shop", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/shop", http.StatusSeeOther)
	}
}

func (env *Environment) RemoveItemRoute(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	user, _ := r.Context().Value(contextKeyCurrentUser).(types.User)

	if r.Method == http.MethodPost {
		idempotencyKey := r.PostForm.Get("_idempotency_key")
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

		tx, err := env.db.Begin()
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec("UPDATE items SET state = 'removed' WHERE id = ?", item.Id)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT INTO item_changes (item_id, user_id, quantity, state, recorded_at) VALUES (?, ?, ?, 'removed', datetime('now'))", item.Id, user.Id, item.Quantity)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec("INSERT INTO idempotency_keys (key, processed_at) VALUES (?, datetime('now'))", idempotencyKey)
		if err != nil {
			if (err.Error() == "UNIQUE constraint failed: idempotency_keys.key") {
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

		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/plan", http.StatusSeeOther)
	}
}
