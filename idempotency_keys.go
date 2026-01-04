package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

func IdempotencyKey() string {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return base64.RawURLEncoding.EncodeToString(key)[0:32]
}

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
