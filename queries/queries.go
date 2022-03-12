package queries

import (
	"database/sql"
	"encoding/json"
	"strings"

	"stravid.com/besserliste/types"
)

type Queries struct {
	getUsers sql.Stmt
	getUserById sql.Stmt
	getCategories sql.Stmt
	getAddedProducts sql.Stmt
	getProducts sql.Stmt
	getDimensions sql.Stmt
	getProduct sql.Stmt
	getAddedItemByProductDimension sql.Stmt
	getAddedItems sql.Stmt
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

	getProducts, err := db.Prepare(`SELECT id, name FROM products ORDER BY name ASC LIMIT 1000`)
	if err != nil {
		panic(err.Error())
	}

	getDimensions, err := db.Prepare(`
		SELECT
			id,
			name,
			json_group_array(json(unit)) AS units
		FROM (
			SELECT
				dimensions.id AS id,
				name,
				json_object(
					'id', units.id,
					'singular_name', units.singular_name,
					'plural_name', units.plural_name,
					'is_base_unit', CASE units.is_base_unit WHEN 1 THEN json('true') ELSE json('false') END,
					'conversion_to_base', units.conversion_to_base,
					'conversion_from_base', units.conversion_from_base
				) AS unit
			FROM dimensions
			INNER JOIN units ON dimensions.id = units.dimension_id
			ORDER BY dimensions.ordering, units.ordering ASC
		)
		GROUP BY id
		LIMIT 10;
	`)
	if err != nil {
		panic(err.Error())
	}

	getProduct, err := db.Prepare(`
		SELECT
			id,
			name,
			json_group_array(json(dimension)) AS dimensions
		FROM (
			SELECT
				product_id AS id,
				product_name AS name,
				json_object(
					'id', dimension_id,
					'name', dimension_name,
					'units', json(units)
				) AS dimension
			FROM (
				SELECT
					product_id,
					product_name,
					dimension_id,
					dimension_name,
					json_group_array(json(unit)) AS units
				FROM (
					SELECT
						products.id AS product_id,
						products.name AS product_name,
						dimensions.id AS dimension_id,
						dimensions.name AS dimension_name,
						json_object(
							'id', units.id,
							'singular_name', units.singular_name,
							'plural_name', units.plural_name,
							'is_base_unit', CASE units.is_base_unit WHEN 1 THEN json('true') ELSE json('false') END,
							'conversion_to_base', units.conversion_to_base,
							'conversion_from_base', units.conversion_from_base
						) AS unit
					FROM products
					INNER JOIN dimensions_products ON products.id = dimensions_products.product_id
					INNER JOIN dimensions ON dimensions_products.dimension_id = dimensions.id
					INNER JOIN units ON dimensions.id = units.dimension_id
					WHERE products.id = ?
					ORDER BY dimensions.ordering, units.ordering ASC
				)
				GROUP BY product_id, product_name, dimension_id, dimension_name
			)
		)
		GROUP BY id, name;
	`)
	if err != nil {
		panic(err.Error())
	}

	getAddedItemByProductDimension, err := db.Prepare(`
		SELECT
			item_id AS id,
			product_name AS name,
			item_quantity AS quantity,
			product_id,
			json_object(
				'id', dimension_id,
				'name', dimension_name,
				'units', json(units)
			) AS dimension
		FROM (
			SELECT
				item_id,
				item_quantity,
				product_id,
				product_name,
				dimension_id,
				dimension_name,
				json_group_array(json(unit)) AS units
			FROM (
				SELECT
					items.id AS item_id,
					items.quantity AS item_quantity,
					products.id AS product_id,
					products.name AS product_name,
					dimensions.id AS dimension_id,
					dimensions.name AS dimension_name,
					json_object(
						'id', units.id,
						'singular_name', units.singular_name,
						'plural_name', units.plural_name,
						'is_base_unit', CASE units.is_base_unit WHEN 1 THEN json('true') ELSE json('false') END,
						'conversion_to_base', units.conversion_to_base,
						'conversion_from_base', units.conversion_from_base
					) AS unit
				FROM items
				INNER JOIN products ON items.product_id = products.id
				INNER JOIN dimensions ON items.dimension_id = dimensions.id
				INNER JOIN units ON dimensions.id = units.dimension_id
				WHERE items.product_id = ? AND items.dimension_id = ? AND items.state = 'added'
				ORDER BY dimensions.ordering, units.ordering ASC
			)
			GROUP BY item_id, item_quantity, product_id, product_name, dimension_id, dimension_name
		);
	`)
	if err != nil {
		panic(err.Error())
	}

	getAddedItems, err := db.Prepare(`
		SELECT
			id,
			name,
			quantity,
			product_id,
			dimension
		FROM (
			SELECT
				item_id AS id,
				product_name AS name,
				item_quantity AS quantity,
				product_id,
				last_change_at,
				json_object(
					'id', dimension_id,
					'name', dimension_name,
					'units', json(units)
				) AS dimension
			FROM (
				SELECT
					item_id,
					item_quantity,
					product_id,
					product_name,
					dimension_id,
					dimension_name,
					MAX(last_change_at) AS last_change_at,
					json_group_array(json(unit)) AS units
				FROM (
					SELECT
						items.id AS item_id,
						items.quantity AS item_quantity,
						products.id AS product_id,
						products.name AS product_name,
						dimensions.id AS dimension_id,
						dimensions.name AS dimension_name,
						item_changes.recorded_at AS last_change_at,
						json_object(
							'id', units.id,
							'singular_name', units.singular_name,
							'plural_name', units.plural_name,
							'is_base_unit', CASE units.is_base_unit WHEN 1 THEN json('true') ELSE json('false') END,
							'conversion_to_base', units.conversion_to_base,
							'conversion_from_base', units.conversion_from_base
						) AS unit
					FROM items
					INNER JOIN products ON items.product_id = products.id
					INNER JOIN dimensions ON items.dimension_id = dimensions.id
					INNER JOIN units ON dimensions.id = units.dimension_id
					INNER JOIN item_changes ON items.id = item_changes.item_id
					WHERE items.state = 'added'
					ORDER BY dimensions.ordering, units.ordering ASC
				)
				GROUP BY item_id, item_quantity, product_id, product_name, dimension_id, dimension_name
			)
		)
		ORDER BY last_change_at DESC
		LIMIT 100
		;
	`)
	if err != nil {
		panic(err.Error())
	}

	return Queries{
		getUsers: *getUsers,
		getUserById: *getUserById,
		getCategories: *getCategories,
		getProducts: *getProducts,
		getDimensions: *getDimensions,
		getProduct: *getProduct,
		getAddedItemByProductDimension: *getAddedItemByProductDimension,
		getAddedItems: *getAddedItems,
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

func (stmt *Queries) GetProducts() ([]types.Product, error) {
	rows, err := stmt.getProducts.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []types.Product{}
	for rows.Next() {
		product := types.Product{}
		err = rows.Scan(&product.Id, &product.Name)
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

func (stmt *Queries) GetDimensions() ([]types.Dimension, error) {
	rows, err := stmt.getDimensions.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dimensions := []types.Dimension{}
	for rows.Next() {
		dimension := types.Dimension{Units: []types.Unit{}}
		var unitJson string

		err = rows.Scan(&dimension.Id, &dimension.Name, &unitJson)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(unitJson), &dimension.Units)
		if err != nil {
			return nil, err
		}

		dimensions = append(dimensions, dimension)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return dimensions, nil
}

func (stmt *Queries) GetProduct(id int) (*types.SelectedProduct, error) {
	row := stmt.getProduct.QueryRow(id)
	product := types.SelectedProduct{}
	var dimensionsJson string
	err := row.Scan(&product.Id, &product.Name, &dimensionsJson)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(dimensionsJson), &product.Dimensions)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (stmt *Queries) GetAddedItemByProductDimension(productId int, dimensionId int) (*types.AddedItem, error) {
	row := stmt.getAddedItemByProductDimension.QueryRow(productId, dimensionId)
	item := types.AddedItem{}
	var dimensionJson string
	err := row.Scan(&item.Id, &item.Name, &item.Quantity, &item.ProductId, &dimensionJson)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(dimensionJson), &item.Dimension)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (stmt *Queries) GetAddedItems() ([]types.AddedItem, error) {
	rows, err := stmt.getAddedItems.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []types.AddedItem{}
	for rows.Next() {
		item := types.AddedItem{}
		var dimensionJson string

		err := rows.Scan(&item.Id, &item.Name, &item.Quantity, &item.ProductId, &dimensionJson)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(dimensionJson), &item.Dimension)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
