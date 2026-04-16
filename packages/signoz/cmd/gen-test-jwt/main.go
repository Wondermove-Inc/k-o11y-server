// gen-test-jwt generates a K-O11y JWT (RS256) for manual SSO testing.
// Usage: go run ./cmd/gen-test-jwt --private-key /path/to/private.pem [--user-id USER] [--tenant-id TENANT] [--role ROLE] [--exp HOURS]
package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	userID := flag.String("user-id", "test-user-001", "K-O11y user ID")
	tenantID := flag.String("tenant-id", "test-tenant-001", "K-O11y tenant (org) ID")
	role := flag.String("role", "admin", "K-O11y role: admin or user")
	privateKeyPath := flag.String("private-key", "", "RSA private key PEM file path (required)")
	issuer := flag.String("issuer", "ko11y", "JWT issuer (must match KO11Y_JWT_ISSUER)")
	expHours := flag.Int("exp", 24, "Token expiration in hours")
	flag.Parse()

	if *privateKeyPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --private-key is required")
		flag.Usage()
		os.Exit(1)
	}

	privateKey, err := loadRSAPrivateKeyFromFile(*privateKeyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading private key: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":   *userID,
		"tenant_id": *tenantID,
		"role":      *role,
		"iss":       *issuer,
		"iat":       now.Unix(),
		"exp":       now.Add(time.Duration(*expHours) * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== K-O11y Test JWT (RS256) ===")
	fmt.Printf("User ID:      %s\n", *userID)
	fmt.Printf("Tenant ID:    %s\n", *tenantID)
	fmt.Printf("Role:         %s\n", *role)
	fmt.Printf("Issuer:       %s\n", *issuer)
	fmt.Printf("Expires:      %s (%dh)\n", now.Add(time.Duration(*expHours)*time.Hour).Format(time.RFC3339), *expHours)
	fmt.Printf("Private Key:  %s\n", *privateKeyPath)
	fmt.Println()
	fmt.Println(tokenString)
}

func loadRSAPrivateKeyFromFile(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// fallback to PKCS1
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return rsaKey, nil
}
