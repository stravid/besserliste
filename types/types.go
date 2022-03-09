package types

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type User struct {
	Id int
	Name string
}

type Category struct {
	Id int
	Name string
}

type AddedProduct struct {
	Id int
	Name string
	Quantity int
	Dimension string
}

type Product struct {
	Id int
	Name string
	SearchName string
	Dimension string
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

func (p *AddedProduct) FormattedQuantity() (string) {
	return FormattedQuantity(p.Quantity, p.Dimension)
}

func FormattedQuantity(quantity int, dimension string) (string) {
	if dimension == "dimensionless" {
		return fmt.Sprintf("%d", quantity)
	} else if dimension == "volume" {
		if quantity >= 1000 {
			return strings.Replace(fmt.Sprintf("%s l", strconv.FormatFloat(float64(quantity) / 1000, 'f', -1, 32)), ".", ",", -1)
		} else {
			return strings.Replace(fmt.Sprintf("%s ml", strconv.FormatFloat(float64(quantity), 'f', -1, 32)), ".", ",", -1)
		}
	} else {
		if quantity >= 1000 {
			return strings.Replace(fmt.Sprintf("%s kg", strconv.FormatFloat(float64(quantity) / 1000, 'f', -1, 32)), ".", ",", -1)
		} else {
			return strings.Replace(fmt.Sprintf("%s g", strconv.FormatFloat(float64(quantity), 'f', -1, 32)), ".", ",", -1)
		}
	}
}
