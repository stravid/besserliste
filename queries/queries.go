package queries

import (
	"database/sql"
	"encoding/json"

	"stravid.com/besserliste/types"
)

type Queries struct {
	getUsers                       sql.Stmt
	getUserById                    sql.Stmt
	getCategories                  sql.Stmt
	getAddedProducts               sql.Stmt
	getProducts                    sql.Stmt
	getDimensions                  sql.Stmt
	getProduct                     sql.Stmt
	getAddedItemByProductDimension sql.Stmt
	getAddedItems                  sql.Stmt
	getRemainingItemsByAlphabet    sql.Stmt
	getRemainingItemsByCategory    sql.Stmt
	getItem                        sql.Stmt
	getGatheredItems               sql.Stmt
	getRemovedItems                sql.Stmt
	getProductByName               sql.Stmt
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

	getProducts, err := db.Prepare(`
		SELECT
			id,
			name_singular,
			name_plural
		FROM products
		ORDER BY name_plural ASC
		LIMIT 1000
	`)
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
					'name_singular', units.name_singular,
					'name_plural', units.name_plural,
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
						products.name_plural AS product_name,
						dimensions.id AS dimension_id,
						dimensions.name AS dimension_name,
						json_object(
							'id', units.id,
							'name_singular', units.name_singular,
							'name_plural', units.name_plural,
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
			product_name_singular AS name_singular,
			product_name_plural AS name_plural,
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
				product_name_singular,
				product_name_plural,
				dimension_id,
				dimension_name,
				json_group_array(json(unit)) AS units
			FROM (
				SELECT
					items.id AS item_id,
					items.quantity AS item_quantity,
					products.id AS product_id,
					products.name_singular AS product_name_singular,
					products.name_plural AS product_name_plural,
					dimensions.id AS dimension_id,
					dimensions.name AS dimension_name,
					json_object(
						'id', units.id,
						'name_singular', units.name_singular,
						'name_plural', units.name_plural,
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
			GROUP BY item_id, item_quantity, product_id, product_name_singular, product_name_plural, dimension_id, dimension_name
		);
	`)
	if err != nil {
		panic(err.Error())
	}

	getAddedItems, err := db.Prepare(`
		SELECT
			id,
			name_singular,
			name_plural,
			quantity,
			product_id,
			dimension
		FROM (
			SELECT
				item_id AS id,
				product_name_singular AS name_singular,
				product_name_plural AS name_plural,
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
					product_name_singular,
					product_name_plural,
					dimension_id,
					dimension_name,
					MAX(last_change_at) AS last_change_at,
					json_group_array(json(unit)) AS units
				FROM (
					SELECT
						items.id AS item_id,
						items.quantity AS item_quantity,
						products.id AS product_id,
						products.name_singular AS product_name_singular,
						products.name_plural AS product_name_plural,
						dimensions.id AS dimension_id,
						dimensions.name AS dimension_name,
						item_changes.recorded_at AS last_change_at,
						json_object(
							'id', units.id,
							'name_singular', units.name_singular,
							'name_plural', units.name_plural,
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
				GROUP BY item_id, item_quantity, product_id, product_name_singular, product_name_plural, dimension_id, dimension_name
			)
		)
		ORDER BY last_change_at DESC
		LIMIT 100
		;
	`)
	if err != nil {
		panic(err.Error())
	}

	getRemainingItemsByAlphabet, err := db.Prepare(`
		SELECT
			id,
			name_singular,
			name_plural,
			quantity,
			product_id,
			dimension
		FROM (
			SELECT
				item_id AS id,
				product_name_singular AS name_singular,
				product_name_plural AS name_plural,
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
					product_name_singular,
					product_name_plural,
					dimension_id,
					dimension_name,
					MAX(last_change_at) AS last_change_at,
					json_group_array(json(unit)) AS units
				FROM (
					SELECT
						items.id AS item_id,
						items.quantity AS item_quantity,
						products.id AS product_id,
						products.name_singular AS product_name_singular,
						products.name_plural AS product_name_plural,
						dimensions.id AS dimension_id,
						dimensions.name AS dimension_name,
						item_changes.recorded_at AS last_change_at,
						json_object(
							'id', units.id,
							'name_singular', units.name_singular,
							'name_plural', units.name_plural,
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
				GROUP BY item_id, item_quantity, product_id, product_name_singular, product_name_plural, dimension_id, dimension_name
			)
		)
		ORDER BY name_plural ASC
		LIMIT 100
		;
	`)
	if err != nil {
		panic(err.Error())
	}

	getRemainingItemsByCategory, err := db.Prepare(`
		SELECT
			id,
			name_singular,
			name_plural,
			quantity,
			product_id,
			dimension
		FROM (
			SELECT
				item_id AS id,
				product_name_singular AS name_singular,
				product_name_plural AS name_plural,
				product_category_id AS category_id,
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
					product_name_singular,
					product_name_plural,
					product_category_id,
					dimension_id,
					dimension_name,
					MAX(last_change_at) AS last_change_at,
					json_group_array(json(unit)) AS units
				FROM (
					SELECT
						items.id AS item_id,
						items.quantity AS item_quantity,
						products.id AS product_id,
						products.name_singular AS product_name_singular,
						products.name_plural AS product_name_plural,
						products.category_id AS product_category_id,
						dimensions.id AS dimension_id,
						dimensions.name AS dimension_name,
						item_changes.recorded_at AS last_change_at,
						json_object(
							'id', units.id,
							'name_singular', units.name_singular,
							'name_plural', units.name_plural,
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
				GROUP BY item_id, item_quantity, product_id, product_name_singular, product_name_plural, product_category_id, dimension_id, dimension_name
			)
		)
		ORDER BY category_id <> ?, name_plural ASC
		LIMIT 100
		;
	`)
	if err != nil {
		panic(err.Error())
	}

	getItem, err := db.Prepare(`
		SELECT
			item_id AS id,
			product_name_singular AS name_singular,
			product_name_plural AS name_plural,
			item_quantity AS quantity,
			item_state AS state,
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
				item_state,
				product_id,
				product_name_singular,
				product_name_plural,
				dimension_id,
				dimension_name,
				json_group_array(json(unit)) AS units
			FROM (
				SELECT
					items.id AS item_id,
					items.quantity AS item_quantity,
					items.state AS item_state,
					products.id AS product_id,
					products.name_singular AS product_name_singular,
					products.name_plural AS product_name_plural,
					dimensions.id AS dimension_id,
					dimensions.name AS dimension_name,
					json_object(
						'id', units.id,
						'name_singular', units.name_singular,
						'name_plural', units.name_plural,
						'conversion_to_base', units.conversion_to_base,
						'conversion_from_base', units.conversion_from_base
					) AS unit
				FROM items
				INNER JOIN products ON items.product_id = products.id
				INNER JOIN dimensions ON items.dimension_id = dimensions.id
				INNER JOIN units ON dimensions.id = units.dimension_id
				WHERE items.id = ?
				ORDER BY dimensions.ordering, units.ordering ASC
			)
			GROUP BY item_id, item_quantity, item_state, product_id, product_name_singular, product_name_plural, dimension_id, dimension_name
		);
	`)
	if err != nil {
		panic(err.Error())
	}

	getGatheredItems, err := db.Prepare(`
		SELECT
			id,
			name_singular,
			name_plural,
			quantity,
			product_id,
			dimension
		FROM (
			SELECT
				item_id AS id,
				product_name_singular AS name_singular,
				product_name_plural AS name_plural,
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
					product_name_singular,
					product_name_plural,
					dimension_id,
					dimension_name,
					MAX(last_change_at) AS last_change_at,
					json_group_array(json(unit)) AS units
				FROM (
					SELECT
						items.id AS item_id,
						items.quantity AS item_quantity,
						products.id AS product_id,
						products.name_singular AS product_name_singular,
						products.name_plural AS product_name_plural,
						dimensions.id AS dimension_id,
						dimensions.name AS dimension_name,
						item_changes.recorded_at AS last_change_at,
						json_object(
							'id', units.id,
							'name_singular', units.name_singular,
							'name_plural', units.name_plural,
							'conversion_to_base', units.conversion_to_base,
							'conversion_from_base', units.conversion_from_base
						) AS unit
					FROM items
					INNER JOIN products ON items.product_id = products.id
					INNER JOIN dimensions ON items.dimension_id = dimensions.id
					INNER JOIN units ON dimensions.id = units.dimension_id
					INNER JOIN item_changes ON items.id = item_changes.item_id
					WHERE items.state = 'gathered'
					ORDER BY dimensions.ordering, units.ordering ASC
				)
				WHERE last_change_at >= datetime('now', '-6 hours')
				GROUP BY item_id, item_quantity, product_id, product_name_singular, product_name_plural, dimension_id, dimension_name
			)
		)
		ORDER BY last_change_at DESC
		LIMIT 100
		;
	`)
	if err != nil {
		panic(err.Error())
	}

	getRemovedItems, err := db.Prepare(`
		SELECT
			id,
			name_singular,
			name_plural,
			quantity,
			product_id,
			dimension
		FROM (
			SELECT
				item_id AS id,
				product_name_singular AS name_singular,
				product_name_plural AS name_plural,
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
					product_name_singular,
					product_name_plural,
					dimension_id,
					dimension_name,
					MAX(last_change_at) AS last_change_at,
					json_group_array(json(unit)) AS units
				FROM (
					SELECT
						items.id AS item_id,
						items.quantity AS item_quantity,
						products.id AS product_id,
						products.name_singular AS product_name_singular,
						products.name_plural AS product_name_plural,
						dimensions.id AS dimension_id,
						dimensions.name AS dimension_name,
						item_changes.recorded_at AS last_change_at,
						json_object(
							'id', units.id,
							'name_singular', units.name_singular,
							'name_plural', units.name_plural,
							'conversion_to_base', units.conversion_to_base,
							'conversion_from_base', units.conversion_from_base
						) AS unit
					FROM items
					INNER JOIN products ON items.product_id = products.id
					INNER JOIN dimensions ON items.dimension_id = dimensions.id
					INNER JOIN units ON dimensions.id = units.dimension_id
					INNER JOIN item_changes ON items.id = item_changes.item_id
					WHERE items.state = 'removed'
					ORDER BY dimensions.ordering, units.ordering ASC
				)
				WHERE last_change_at >= datetime('now', '-6 hours')
				GROUP BY item_id, item_quantity, product_id, product_name_singular, product_name_plural, dimension_id, dimension_name
			)
		)
		ORDER BY last_change_at DESC
		LIMIT 20
		;
	`)
	if err != nil {
		panic(err.Error())
	}

	getProductByName, err := db.Prepare(`
		SELECT
			id,
			name_singular,
			name_plural
		FROM products
		WHERE lower(name_singular, 'de_AT') = lower(?) OR lower(name_plural, 'de_AT') = lower(?)
		LIMIT 1
	`)
	if err != nil {
		panic(err.Error())
	}

	return Queries{
		getUsers:                       *getUsers,
		getUserById:                    *getUserById,
		getCategories:                  *getCategories,
		getProducts:                    *getProducts,
		getDimensions:                  *getDimensions,
		getProduct:                     *getProduct,
		getAddedItemByProductDimension: *getAddedItemByProductDimension,
		getAddedItems:                  *getAddedItems,
		getRemainingItemsByAlphabet:    *getRemainingItemsByAlphabet,
		getRemainingItemsByCategory:    *getRemainingItemsByCategory,
		getItem:                        *getItem,
		getGatheredItems:               *getGatheredItems,
		getRemovedItems:                *getRemovedItems,
		getProductByName:               *getProductByName,
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
		p := types.Product{}
		err = rows.Scan(&p.Id, &p.NameSingular, &p.NamePlural)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
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
	i := types.AddedItem{}
	var dimensionJson string
	err := row.Scan(&i.Id, &i.NameSingular, &i.NamePlural, &i.Quantity, &i.ProductId, &dimensionJson)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(dimensionJson), &i.Dimension)
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func (stmt *Queries) GetAddedItems() ([]types.AddedItem, error) {
	rows, err := stmt.getAddedItems.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []types.AddedItem{}
	for rows.Next() {
		i := types.AddedItem{}
		var dimensionJson string

		err := rows.Scan(&i.Id, &i.NameSingular, &i.NamePlural, &i.Quantity, &i.ProductId, &dimensionJson)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(dimensionJson), &i.Dimension)
		if err != nil {
			return nil, err
		}

		items = append(items, i)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (stmt *Queries) GetRemainingItemsByAlphabet() ([]types.AddedItem, error) {
	rows, err := stmt.getRemainingItemsByAlphabet.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []types.AddedItem{}
	for rows.Next() {
		i := types.AddedItem{}
		var dimensionJson string

		err := rows.Scan(&i.Id, &i.NameSingular, &i.NamePlural, &i.Quantity, &i.ProductId, &dimensionJson)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(dimensionJson), &i.Dimension)
		if err != nil {
			return nil, err
		}

		items = append(items, i)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (stmt *Queries) GetRemainingItemsByCategory(id int) ([]types.AddedItem, error) {
	rows, err := stmt.getRemainingItemsByCategory.Query(id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []types.AddedItem{}
	for rows.Next() {
		i := types.AddedItem{}
		var dimensionJson string

		err := rows.Scan(&i.Id, &i.NameSingular, &i.NamePlural, &i.Quantity, &i.ProductId, &dimensionJson)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(dimensionJson), &i.Dimension)
		if err != nil {
			return nil, err
		}

		items = append(items, i)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (stmt *Queries) GetItem(itemId int) (*types.SelectedItem, error) {
	row := stmt.getItem.QueryRow(itemId)
	i := types.SelectedItem{}
	var dimensionJson string
	err := row.Scan(&i.Id, &i.NameSingular, &i.NamePlural, &i.Quantity, &i.State, &i.ProductId, &dimensionJson)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(dimensionJson), &i.Dimension)
	if err != nil {
		return nil, err
	}

	return &i, nil
}

func (stmt *Queries) GetGatheredItems() ([]types.AddedItem, error) {
	rows, err := stmt.getGatheredItems.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []types.AddedItem{}
	for rows.Next() {
		i := types.AddedItem{}
		var dimensionJson string

		err := rows.Scan(&i.Id, &i.NameSingular, &i.NamePlural, &i.Quantity, &i.ProductId, &dimensionJson)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(dimensionJson), &i.Dimension)
		if err != nil {
			return nil, err
		}

		items = append(items, i)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (stmt *Queries) GetRemovedItems() ([]types.AddedItem, error) {
	rows, err := stmt.getRemovedItems.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []types.AddedItem{}
	for rows.Next() {
		i := types.AddedItem{}
		var dimensionJson string

		err := rows.Scan(&i.Id, &i.NameSingular, &i.NamePlural, &i.Quantity, &i.ProductId, &dimensionJson)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(dimensionJson), &i.Dimension)
		if err != nil {
			return nil, err
		}

		items = append(items, i)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (stmt *Queries) GetProductByName(name string) (*types.Product, error) {
	row := stmt.getProductByName.QueryRow(name, name)
	p := types.Product{}
	err := row.Scan(&p.Id, &p.NameSingular, &p.NamePlural)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
