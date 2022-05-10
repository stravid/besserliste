package queries

import (
	"database/sql"
	"encoding/json"
	"errors"
	"embed"
	"strings"

	"stravid.com/besserliste/types"
)

//go:embed *.sql
var files embed.FS

type Queries struct {
	statements map[string]*sql.Stmt
}

func Build(db *sql.DB) Queries {
	statements := make(map[string]*sql.Stmt)
	queryDirectoryEntries, err := files.ReadDir(".")
	if err != nil {
		panic(err.Error())
	}
	for _, entry := range queryDirectoryEntries {
		sql, err := files.ReadFile(entry.Name())
		if err != nil {
			panic(err.Error())
		}
		stmt, err := db.Prepare(string(sql))
		if err != nil {
			panic(err.Error())
		}

		statements[strings.ReplaceAll(entry.Name(), ".sql", "")] = stmt
	}

	return Queries{
		statements: statements,
	}
}

func (stmt *Queries) GetCategories(tx *sql.Tx) ([]types.Category, error) {
	if _, ok := stmt.statements["GetCategories"]; !ok {
		return nil, errors.New("Unknown query `GetCategories`")
	}

	rows, err := tx.Stmt(stmt.statements["GetCategories"]).Query()
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

func (stmt *Queries) GetUserById(tx *sql.Tx, id int) (*types.User, error) {
	if _, ok := stmt.statements["GetUserById"]; !ok {
		return nil, errors.New("Unknown query `GetUserById`")
	}

	row := tx.Stmt(stmt.statements["GetUserById"]).QueryRow(id)
	user := types.User{}
	err := row.Scan(&user.Id, &user.Name)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (stmt *Queries) GetUsers(tx *sql.Tx) ([]types.User, error) {
	if _, ok := stmt.statements["GetUsers"]; !ok {
		return nil, errors.New("Unknown query `GetUsers`")
	}

	rows, err := tx.Stmt(stmt.statements["GetUsers"]).Query()
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

func (stmt *Queries) GetProducts(tx *sql.Tx) ([]types.Product, error) {
	if _, ok := stmt.statements["GetProducts"]; !ok {
		return nil, errors.New("Unknown query `GetProducts`")
	}

	rows, err := tx.Stmt(stmt.statements["GetProducts"]).Query()
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

func (stmt *Queries) GetDimensions(tx *sql.Tx) ([]types.Dimension, error) {
	if _, ok := stmt.statements["GetDimensions"]; !ok {
		return nil, errors.New("Unknown query `GetDimensions`")
	}

	rows, err := tx.Stmt(stmt.statements["GetDimensions"]).Query()
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

func (stmt *Queries) GetProduct(tx *sql.Tx, id int) (*types.SelectedProduct, error) {
	if _, ok := stmt.statements["GetProduct"]; !ok {
		return nil, errors.New("Unknown query `GetProduct`")
	}

	row := tx.Stmt(stmt.statements["GetProduct"]).QueryRow(id)
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

func (stmt *Queries) GetAddedItemByProductDimension(tx *sql.Tx, productId int, dimensionId int) (*types.AddedItem, error) {
	if _, ok := stmt.statements["GetAddedItemByProductDimension"]; !ok {
		return nil, errors.New("Unknown query `GetAddedItemByProductDimension`")
	}

	row := tx.Stmt(stmt.statements["GetAddedItemByProductDimension"]).QueryRow(productId, dimensionId)
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

func (stmt *Queries) GetAddedItems(tx *sql.Tx) ([]types.AddedItem, error) {
	if _, ok := stmt.statements["GetAddedItems"]; !ok {
		return nil, errors.New("Unknown query `GetAddedItems`")
	}

	rows, err := tx.Stmt(stmt.statements["GetAddedItems"]).Query()
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

func (stmt *Queries) GetRemainingItemsByAlphabet(tx *sql.Tx) ([]types.AddedItem, error) {
	if _, ok := stmt.statements["GetRemainingItemsByAlphabet"]; !ok {
		return nil, errors.New("Unknown query `GetRemainingItemsByAlphabet`")
	}

	rows, err := tx.Stmt(stmt.statements["GetRemainingItemsByAlphabet"]).Query()
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

func (stmt *Queries) GetRemainingItemsByCategory(tx *sql.Tx, id int) ([]types.AddedItem, error) {
	if _, ok := stmt.statements["GetRemainingItemsByCategory"]; !ok {
		return nil, errors.New("Unknown query `GetRemainingItemsByCategory`")
	}

	rows, err := tx.Stmt(stmt.statements["GetRemainingItemsByCategory"]).Query(id)
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

func (stmt *Queries) GetItem(tx *sql.Tx, itemId int) (*types.SelectedItem, error) {
	if _, ok := stmt.statements["GetItem"]; !ok {
		return nil, errors.New("Unknown query `GetItem`")
	}

	row := tx.Stmt(stmt.statements["GetItem"]).QueryRow(itemId)
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

func (stmt *Queries) GetGatheredItems(tx *sql.Tx) ([]types.AddedItem, error) {
	if _, ok := stmt.statements["GetGatheredItems"]; !ok {
		return nil, errors.New("Unknown query `GetGatheredItems`")
	}

	rows, err := tx.Stmt(stmt.statements["GetGatheredItems"]).Query()
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

func (stmt *Queries) GetRemovedItems(tx *sql.Tx) ([]types.AddedItem, error) {
	if _, ok := stmt.statements["GetRemovedItems"]; !ok {
		return nil, errors.New("Unknown query `GetRemovedItems`")
	}

	rows, err := tx.Stmt(stmt.statements["GetRemovedItems"]).Query()
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

func (stmt *Queries) GetProductByName(tx *sql.Tx, name string) (*types.Product, error) {
	if _, ok := stmt.statements["GetProductByName"]; !ok {
		return nil, errors.New("Unknown query `GetProductByName`")
	}

	row := tx.Stmt(stmt.statements["GetProductByName"]).QueryRow(name, name)
	p := types.Product{}
	err := row.Scan(&p.Id, &p.NameSingular, &p.NamePlural)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (stmt *Queries) SetItemState(tx *sql.Tx, itemId int, state string) (error) {
	if _, ok := stmt.statements["SetItemState"]; !ok {
		return errors.New("Unknown query `SetItemState`")
	}

	_, err := tx.Stmt(stmt.statements["SetItemState"]).Exec(state, itemId)
	return err
}

func (stmt *Queries) InsertItemChange(tx *sql.Tx, itemId int64, userId int, dimensionId int, quantity int64, state string) (error) {
	if _, ok := stmt.statements["InsertItemChange"]; !ok {
		return errors.New("Unknown query `InsertItemChange`")
	}

	_, err := tx.Stmt(stmt.statements["InsertItemChange"]).Exec(itemId, userId, dimensionId, quantity, state)
	return err
}

func (stmt *Queries) InsertIdempotencyKey(tx *sql.Tx, idempotencyKey string) (error) {
	if _, ok := stmt.statements["InsertIdempotencyKey"]; !ok {
		return errors.New("Unknown query `InsertIdempotencyKey`")
	}

	_, err := tx.Stmt(stmt.statements["InsertIdempotencyKey"]).Exec(idempotencyKey)
	return err
}

func (stmt *Queries) SetItemQuantity(tx *sql.Tx, itemId int64, quantity int64) (error) {
	if _, ok := stmt.statements["SetItemQuantity"]; !ok {
		return errors.New("Unknown query `SetItemQuantity`")
	}

	_, err := tx.Stmt(stmt.statements["SetItemQuantity"]).Exec(quantity, itemId)
	return err
}

func (stmt *Queries) SetItemQuantityForDifferentDimension(tx *sql.Tx, itemId int, quantity int64, dimensionId int) (error) {
	if _, ok := stmt.statements["SetItemQuantityForDifferentDimension"]; !ok {
		return errors.New("Unknown query `SetItemQuantityForDifferentDimension`")
	}

	_, err := tx.Stmt(stmt.statements["SetItemQuantityForDifferentDimension"]).Exec(quantity, dimensionId, itemId)
	return err
}

func (stmt *Queries) InsertProduct(tx *sql.Tx, nameSingular string, namePlural string) (sql.Result, error) {
	if _, ok := stmt.statements["InsertProduct"]; !ok {
		return nil, errors.New("Unknown query `InsertProduct`")
	}

	return tx.Stmt(stmt.statements["InsertProduct"]).Exec(nameSingular, namePlural)
}

func (stmt *Queries) InsertProductDimension(tx *sql.Tx, productId int64, dimensionId string) (sql.Result, error) {
	if _, ok := stmt.statements["InsertProductDimension"]; !ok {
		return nil, errors.New("Unknown query `InsertProductDimension`")
	}

	return tx.Stmt(stmt.statements["InsertProductDimension"]).Exec(dimensionId, productId)
}

func (stmt *Queries) InsertProductCategory(tx *sql.Tx, productId int64, categoryId string) (sql.Result, error) {
	if _, ok := stmt.statements["InsertProductCategory"]; !ok {
		return nil, errors.New("Unknown query `InsertProductCategory`")
	}

	return tx.Stmt(stmt.statements["InsertProductCategory"]).Exec(categoryId, productId)
}

func (stmt *Queries) InsertProductChange(tx *sql.Tx, productId int64, userId int, nameSingular string, namePlural string) (sql.Result, error) {
	if _, ok := stmt.statements["InsertProductChange"]; !ok {
		return nil, errors.New("Unknown query `InsertProductChange`")
	}

	return tx.Stmt(stmt.statements["InsertProductChange"]).Exec(productId, userId, nameSingular, namePlural)
}

func (stmt *Queries) InsertItem(tx *sql.Tx, productId int, dimensionId int, quantity int64) (sql.Result, error) {
	if _, ok := stmt.statements["InsertItem"]; !ok {
		return nil, errors.New("Unknown query `InsertItem`")
	}

	return tx.Stmt(stmt.statements["InsertItem"]).Exec(productId, dimensionId, quantity)
}
func (stmt *Queries) RemovePreviousIdempotencyKeys(tx *sql.Tx) (sql.Result, error) {
	if _, ok := stmt.statements["RemovePreviousIdempotencyKeys"]; !ok {
		return nil, errors.New("Unknown query `RemovePreviousIdempotencyKeys`")
	}

	return tx.Stmt(stmt.statements["RemovePreviousIdempotencyKeys"]).Exec()
}
