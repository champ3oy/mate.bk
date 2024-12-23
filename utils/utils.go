package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func ConvertCurrencyToFloat(currencyStr string) (float64, error) {
	// Allow for both "GHS" prefixed strings and plain numeric strings
	if !strings.HasPrefix(currencyStr, "GHS") && !isNumeric(currencyStr) {
		return 0, nil
	}

	// If it has the "GHS" prefix, trim it
	if strings.HasPrefix(currencyStr, "GHS") {
		cleanedStr := strings.TrimPrefix(currencyStr, "GHS")
		cleanedStr = strings.TrimSpace(cleanedStr)
		value, err := parseFloat(cleanedStr)
		if err != nil {
			return 0, nil
		}
		return value, nil
	}

	// If it's a plain numeric string, parse it directly
	value, err := parseFloat(currencyStr)
	if err != nil {
		return 0, nil
	}
	return value, nil
}

// Helper function to check if a string is numeric
func isNumeric(str string) bool {
	_, err := strconv.ParseFloat(str, 64)
	return err == nil
}

// Helper function to parse float and handle errors
func parseFloat(cleanedStr string) (float64, error) {
	value, err := strconv.ParseFloat(cleanedStr, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing currency value: %s", cleanedStr)
	}
	if value != value {
		return 0, fmt.Errorf("invalid currency value: %s", cleanedStr)
	}
	return value, nil
}
