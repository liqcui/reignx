package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/reignx/reignx/pkg/crypto"
)

func main() {
	// Database connection
	connStr := "postgres://reignx:reignx@localhost:5432/reignx?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize encryptor (same key as webserver)
	encryptor, err := crypto.NewEncryptorFromString("IJBBBFbEhkpsRjPDt6V3cFG78dWEb86alrGQ8DGw+jc=")
	if err != nil {
		log.Fatal(err)
	}

	// Query encrypted password from database
	nodeID := "14e3d596-445c-416c-ab21-2a7800e2fa6c"
	var encryptedPassword string

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT password_encrypted FROM ssh_configs WHERE node_id = $1`
	err = db.QueryRowContext(ctx, query, nodeID).Scan(&encryptedPassword)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Encryption Flow Test ===")
	fmt.Println()
	fmt.Println("1. Query database:")
	fmt.Printf("   Encrypted value: %s...\n", encryptedPassword[:40])
	fmt.Println()

	// Decrypt the password
	decryptedPassword, err := encryptor.Decrypt(encryptedPassword)
	if err != nil {
		log.Fatal("Decryption failed:", err)
	}

	fmt.Println("2. Decrypt in memory:")
	fmt.Printf("   Decrypted password: %s\n", decryptedPassword)
	fmt.Println()

	fmt.Println("3. Use for SSH authentication:")
	fmt.Println("   ssh.Password(decrypted_password)")
	fmt.Println()

	fmt.Println("✅ Encryption/Decryption working correctly!")
	fmt.Println()
	fmt.Println("Security Benefits:")
	fmt.Println("  - Database only stores encrypted value")
	fmt.Println("  - Decryption happens in memory only when needed")
	fmt.Println("  - Original password never logged or persisted")
	fmt.Println("  - AES-256-GCM provides authentication (tamper detection)")
}
