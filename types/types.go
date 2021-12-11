package types

import (
	"errors"
	"fmt"
	"strconv"
)

type User struct {
	Id int
	Name string
}

func UserIdFromString(input string) (*int, error) {
	id, err := strconv.Atoi(input)
	if err != nil {
		return nil, err
	}

	if id < 1 {
		return nil, errors.New(fmt.Sprintf("User id cannot be zero or less, got %v", id))
	}

	return &id, nil
}
