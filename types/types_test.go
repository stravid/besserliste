package types

import (
    "testing"
)

func TestFormattedQuantity(t *testing.T) {
    if r := FormattedQuantity(1, "dimensionless"); r != "1" {
        t.Fatalf("dimensionless is not formatted (%s instead of %s)", r, "1")
    }
    if r := FormattedQuantity(1000, "volume"); r != "1 l" {
        t.Fatalf("one liter (%s instead of %s)", r, "1 l")
    }
    if r := FormattedQuantity(330, "volume"); r != "330 ml" {
        t.Fatalf("less than a liter (%s instead of %s)", r, "330 ml")
    }
    if r := FormattedQuantity(1250, "volume"); r != "1,25 l" {
        t.Fatalf("more than a liter (%s instead of %s)", r, "1,25 l")
    }
    if r := FormattedQuantity(1000, "mass"); r != "1 kg" {
        t.Fatalf("one kg (%s instead of %s)", r, "1 kg")
    }
    if r := FormattedQuantity(200, "mass"); r != "200 g" {
        t.Fatalf("less than a kg (%s instead of %s)", r, "200 g")
    }
    if r := FormattedQuantity(1234, "mass"); r != "1,234 kg" {
        t.Fatalf("more than a kg (%s instead of %s)", r, "1,234 kg")
    }
}
