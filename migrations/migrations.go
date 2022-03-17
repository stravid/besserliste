package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
)

//go:embed *.sql
var files embed.FS

func Run(db *sql.DB) {
	version := getVersion(db)

	for {
		version++
		path := fmt.Sprintf("%v.sql", version)
		migration, err := files.ReadFile(path)

		if err != nil {
			break
		}

		query := fmt.Sprintf(`
			BEGIN;
			%v
			PRAGMA user_version = %v;
			COMMIT;
		`,
			string(migration),
			version)

		_, err = db.Exec(query)

		if err != nil {
			log.Panicln(fmt.Sprintf("Migration `migrations/%v` failed:", path), err.Error())
		}

		log.Println(fmt.Sprintf("Ran migration %v", path))
	}
}

func getVersion(db *sql.DB) int {
	row := db.QueryRow(`PRAGMA user_version;`)
	var version int

	err := row.Scan(&version)
	if err != nil {
		log.Panicln("Cannot get database user version", err)
	}

	return version
}
