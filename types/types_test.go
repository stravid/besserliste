package types

import (
	"testing"
)

func TestFormattedQuantity(t *testing.T) {
    dimensionlessUnits := []Unit{
        {
            Id: 1,
            NameSingular: "Flasche",
            NamePlural: "Flaschen",
            IsBaseUnit: true,
            ConversionToBase: 1,
            ConversionFromBase: 1,
        },
    }

    volumeUnits := []Unit{
        {
            Id: 2,
            NameSingular: "ml",
            NamePlural: "ml",
            IsBaseUnit: true,
            ConversionToBase: 1,
            ConversionFromBase: 1,
        },
        {
            Id: 3,
            NameSingular: "l",
            NamePlural: "l",
            IsBaseUnit: false,
            ConversionToBase: 1000,
            ConversionFromBase: 0.001,
        },
    }

    if r := FormattedQuantity(1, dimensionlessUnits); r != "1 Flasche" {
        t.Fatalf("%s instead of %s", r, "1 Flasche")
    }

    if r := FormattedQuantity(3, dimensionlessUnits); r != "3 Flaschen" {
        t.Fatalf("%s instead of %s", r, "3 Flaschen")
    }

    if r := FormattedQuantity(1, volumeUnits); r != "1 ml" {
        t.Fatalf("%s instead of %s", r, "1 ml")
    }

    if r := FormattedQuantity(33, volumeUnits); r != "33 ml" {
        t.Fatalf("%s instead of %s", r, "33 ml")
    }

    if r := FormattedQuantity(1000, volumeUnits); r != "1 l" {
        t.Fatalf("%s instead of %s", r, "1 l")
    }

    if r := FormattedQuantity(1250, volumeUnits); r != "1,25 l" {
        t.Fatalf("%s instead of %s", r, "1,25 l")
    }
}
