package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/reignx/reignx/pkg/crypto"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Encrypt sensitive data",
	Long:  "Encrypt passwords and other sensitive data for secure storage",
}

var encryptPasswordCmd = &cobra.Command{
	Use:   "password",
	Short: "Encrypt a password",
	Long:  "Encrypt a password using AES-256-GCM encryption",
	Run:   runEncryptPassword,
}

var encryptKeyGenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate encryption key",
	Long:  "Generate a random 32-byte AES-256 encryption key",
	Run:   runKeyGen,
}

var (
	encryptionKey string
	password      string
	silent        bool
)

func init() {
	encryptCmd.AddCommand(encryptPasswordCmd)
	encryptCmd.AddCommand(encryptKeyGenCmd)

	encryptPasswordCmd.Flags().StringVarP(&encryptionKey, "key", "k", "", "Encryption key (32-byte base64 encoded or plain string)")
	encryptPasswordCmd.Flags().StringVarP(&password, "password", "p", "", "Password to encrypt (will prompt if not provided)")
	encryptPasswordCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Silent mode - only output encrypted password")

	encryptPasswordCmd.MarkFlagRequired("key")
}

func runEncryptPassword(cmd *cobra.Command, args []string) {
	// Get password from flag or prompt
	plainPassword := password
	if plainPassword == "" {
		fmt.Print("Enter password to encrypt: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
		plainPassword = string(passwordBytes)
		fmt.Println() // New line after password input
	}

	if plainPassword == "" {
		fmt.Fprintf(os.Stderr, "Error: password cannot be empty\n")
		os.Exit(1)
	}

	// Create encryptor
	encryptor, err := crypto.NewEncryptorFromString(encryptionKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating encryptor: %v\n", err)
		os.Exit(1)
	}

	// Encrypt password
	encrypted, err := encryptor.Encrypt(plainPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encrypting password: %v\n", err)
		os.Exit(1)
	}

	if silent {
		fmt.Println(encrypted)
	} else {
		fmt.Println("\n✅ Password encrypted successfully!")
		fmt.Println("\nEncrypted password:")
		fmt.Printf("  %s\n\n", encrypted)
		fmt.Println("Store this encrypted value in your configuration or database.")
		fmt.Println("The encryption key must be kept secure and provided to the application.")
	}
}

func runKeyGen(cmd *cobra.Command, args []string) {
	key, err := crypto.GenerateKeyString()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Encryption key generated successfully!")
	fmt.Println("\nBase64-encoded key (32 bytes for AES-256):")
	fmt.Printf("  %s\n\n", key)
	fmt.Println("⚠️  IMPORTANT:")
	fmt.Println("  - Store this key securely (environment variable, secret manager, etc.)")
	fmt.Println("  - Never commit this key to version control")
	fmt.Println("  - Use the same key for encryption and decryption")
	fmt.Println("  - If you lose this key, encrypted data cannot be recovered")
	fmt.Println("\nUsage in config file:")
	fmt.Println("  security:")
	fmt.Printf("    encryption_key: \"%s\"\n\n", key)
	fmt.Println("Usage as environment variable:")
	fmt.Printf("  export REIGNX_ENCRYPTION_KEY=\"%s\"\n", key)
}

func promptForConfirmation(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/n): ", message)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
