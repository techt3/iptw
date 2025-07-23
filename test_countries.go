package main

import (
	"fmt"
	"log"

	"iptw/internal/resources"
)

func main() {
	// Test GetAlpha2ByName
	fmt.Println("Testing GetAlpha2ByName:")

	testCases := []string{
		"United States of America",
		"Afghanistan",
		"Albania",
		"canada", // case insensitive test
	}

	for _, country := range testCases {
		alpha2, err := resources.GetAlpha2ByName(country)
		if err != nil {
			fmt.Printf("  %s: Error - %v\n", country, err)
		} else {
			fmt.Printf("  %s: %s\n", country, alpha2)
		}
	}

	fmt.Println("\nTesting GetNameByAlpha2:")

	testCodes := []string{"US", "AF", "AL", "ca"} // case insensitive test

	for _, code := range testCodes {
		name, err := resources.GetNameByAlpha2(code)
		if err != nil {
			fmt.Printf("  %s: Error - %v\n", code, err)
		} else {
			fmt.Printf("  %s: %s\n", code, name)
		}
	}

	fmt.Println("\nTesting GetCountryByAlpha2:")
	country, err := resources.GetCountryByAlpha2("US")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("  US: Name=%s, Alpha3=%s, Region=%s\n", country.Name, country.Alpha3, country.Region)
	}

	fmt.Println("\nTotal countries loaded:")
	countries := resources.GetAllCountries()
	fmt.Printf("  %d countries\n", len(countries))
}
