// Script to generate Android API constants from Go constants
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	// Read the Go constants file
	file, err := os.Open("lib/api/constants.go")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Create the Kotlin output file
	outFile, err := os.Create("android/app/src/main/java/com/syncthing/android/util/ApiConstants.kt")
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer outFile.Close()

	// Write the header
	header := `package com.syncthing.android.util

/**
 * Shared constants between desktop and Android versions of Syncthing
 * These are automatically generated from lib/api/constants.go
 * DO NOT EDIT MANUALLY - Run 'go run script/generate-android-constants.go' to regenerate
 */
object ApiConstants {
`
	outFile.WriteString(header)

	// Regular expressions to match const declarations
	constRegex := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*=\s*("[^"]*"|\d+)`)
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		
		// Check if this is a const declaration
		matches := constRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			name := matches[1]
			value := matches[2]
			
			// Convert Go constant name to Kotlin constant name
			kotlinName := convertToKotlinConstantName(name)
			
			// Write to Kotlin file
			if strings.HasPrefix(value, "\"") {
				// String constant
				outFile.WriteString(fmt.Sprintf("    const val %s = %s\n", kotlinName, value))
			} else {
				// Numeric constant
				outFile.WriteString(fmt.Sprintf("    const val %s = %s\n", kotlinName, value))
			}
		}
	}

	// Write the footer
	footer := `}
`
	outFile.WriteString(footer)

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
	}
	
	fmt.Println("Android API constants generated successfully!")
}

// Convert Go constant name to Kotlin constant name
// e.g., SystemStatusEndpoint -> SYSTEM_STATUS_ENDPOINT
func convertToKotlinConstantName(goName string) string {
	var result strings.Builder
	for i, r := range goName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToUpper(result.String())
}