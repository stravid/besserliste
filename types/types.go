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

type Product struct {
	Id int
	NameSingular string
	NamePlural string
}

type Dimension struct {
	Id int `json:"id"`
	Name string `json:"name"`
	Units []Unit `json:"units"`
}

type Unit struct {
	Id int `json:"id"`
	NameSingular string `json:"name_singular"`
	NamePlural string `json:"name_plural"`
	IsBaseUnit bool `json:"is_base_unit"`
	ConversionToBase float64 `json:"conversion_to_base"`
	ConversionFromBase float64 `json:"conversion_from_base"`
}

type SelectedProduct struct {
	Id int `json:"id"`
	Name string `json:"name"`
	Dimensions []Dimension `json:"dimensions"`
}

type AddedItem struct {
	Id int `json:"id"`
	NameSingular string `json:"name_singular"`
	NamePlural string `json:"name_plural"`
	Quantity int
	ProductId int
	Dimension Dimension `json:"dimension"`
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

func (i *AddedItem) FormattedQuantity() (string) {
	return FormattedQuantity(i.Quantity, i.Dimension.Units)
}

func FormattedQuantity(quantity int, units []Unit) (string) {
	floatQuantity := float64(quantity)
	var bestFittingUnit Unit

	for _, unit := range units {
		if bestFittingUnit.Id == 0 {
			bestFittingUnit = unit
		} else {
			newUnitQuantityIsSmaller := bestFittingUnit.ConversionFromBase * floatQuantity > unit.ConversionFromBase * floatQuantity
			newUnitQuantityIsBigEnough := unit.ConversionFromBase * floatQuantity >= 1
			if newUnitQuantityIsSmaller && newUnitQuantityIsBigEnough {
				bestFittingUnit = unit
			}

		}
	}

	formattedQuantity := strings.Replace(strconv.FormatFloat(bestFittingUnit.ConversionFromBase * floatQuantity, 'f', -1, 32), ".", ",", -1)

	if bestFittingUnit.ConversionFromBase * floatQuantity > 1 {
		return fmt.Sprintf("%s %s", formattedQuantity, bestFittingUnit.NamePlural)
	} else {
		return fmt.Sprintf("%s %s", formattedQuantity, bestFittingUnit.NameSingular)
	}
}

func (i *AddedItem) FormattedName() (string) {
	if i.Quantity == 1 {
		return i.NameSingular
	} else {
		return i.NamePlural
	}
}

func (p *Product) SearchTerm() (string) {
	if p.NameSingular != p.NamePlural {
		return strings.ToLower(fmt.Sprintf("%s %s", p.NameSingular, p.NamePlural))
	} else {
		return strings.ToLower(p.NamePlural)
	}
}
