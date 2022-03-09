package queries

import (
	"database/sql"
	"strings"

	"stravid.com/besserliste/types"
)

type Queries struct {
	getUsers sql.Stmt
	getUserById sql.Stmt
	getCategories sql.Stmt
	getAddedProducts sql.Stmt
	getProducts sql.Stmt
}

func Build(db *sql.DB) Queries {
	getUsers, err := db.Prepare(`SELECT id, name FROM users ORDER BY name ASC LIMIT 10`)
	if err != nil {
		panic(err.Error())
	}

	getUserById, err := db.Prepare(`SELECT id, name FROM users WHERE id = ? LIMIT 1`)
	if err != nil {
		panic(err.Error())
	}

	getCategories, err := db.Prepare(`SELECT id, name FROM categories ORDER BY ordering ASC LIMIT 10`)
	if err != nil {
		panic(err.Error())
	}

	getAddedProducts, err := db.Prepare(`
		SELECT
			id,
			name,
			quantity,
			dimension
		FROM (
			SELECT
				product_id AS id,
				name,
				SUM(items.quantity) AS quantity,
				dimension,
				MAX(recorded_at) AS last_change_at
			FROM items
			INNER JOIN products ON items.product_id = products.id
			INNER JOIN item_changes ON items.id = item_changes.item_id
			WHERE items.state = 'added'
			GROUP BY product_id
		)
		ORDER BY last_change_at DESC
		LIMIT 100
	`)
	if err != nil {
		panic(err.Error())
	}

	getProducts, err := db.Prepare(`SELECT id, name, dimension FROM products ORDER BY name ASC LIMIT 1000`)
	if err != nil {
		panic(err.Error())
	}

	return Queries{
		getUsers: *getUsers,
		getUserById: *getUserById,
		getCategories: *getCategories,
		getAddedProducts: *getAddedProducts,
		getProducts: *getProducts,
	}
}

func (stmt *Queries) GetCategories() ([]types.Category, error) {
	rows, err := stmt.getCategories.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []types.Category{}
	for rows.Next() {
		category := types.Category{}
		err = rows.Scan(&category.Id, &category.Name)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func (stmt *Queries) GetUserById(id int) (*types.User, error) {
	row := stmt.getUserById.QueryRow(id)
	user := types.User{}
	err := row.Scan(&user.Id, &user.Name)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (stmt *Queries) GetUsers() ([]types.User, error) {
	rows, err := stmt.getUsers.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []types.User{}
	for rows.Next() {
		user := types.User{}
		err = rows.Scan(&user.Id, &user.Name)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (stmt *Queries) GetAddedProducts() ([]types.AddedProduct, error) {
	rows, err := stmt.getAddedProducts.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []types.AddedProduct{}
	for rows.Next() {
		product := types.AddedProduct{}
		err = rows.Scan(&product.Id, &product.Name, &product.Quantity, &product.Dimension)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func (stmt *Queries) GetProducts() ([]types.Product, error) {
	rows, err := stmt.getProducts.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []types.Product{}
	for rows.Next() {
		product := types.Product{}
		err = rows.Scan(&product.Id, &product.Name, &product.Dimension)
		if err != nil {
			return nil, err
		}
		product.SearchName = strings.ToLower(product.Name)
		products = append(products, product)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
