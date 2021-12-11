package queries

import (
	"database/sql"
	"stravid.com/besserliste/types"
)

type Queries struct {
	getUsers sql.Stmt
	getUserById sql.Stmt
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

	return Queries{
		getUsers: *getUsers,
		getUserById: *getUserById,
	}
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

func (stmt *Queries) GetUserById(id int) (*types.User, error) {
	row := stmt.getUserById.QueryRow(id)
	user := types.User{}
	err := row.Scan(&user.Id, &user.Name)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
