package blockchain

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestUploadPDF(t *testing.T) {
	// Load .env
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Log("Warning: could not load .env file:", err)
	}

	jwt := os.Getenv("PINATA_JWT")
	if jwt == "" {
		t.Fatal("PINATA_JWT environment variable is empty!")
	}
	t.Log("PINATA_JWT length:", len(jwt))
	t.Log("PINATA_JWT prefix:", jwt[:30]+"...")

	client := &PinataClient{apiKey: jwt}
	
	// Create simple dummy PDF bytes
	mockPDF := []byte("%PDF-1.4 dummy pdf content")

	cid, err := client.UploadPDF(mockPDF, "test_cert_go.pdf")
	if err != nil {
		log.Printf("Upload failed: %v", err)
		t.Fatalf("Failed to upload: %v", err)
	}

	t.Log("Successfully uploaded! CID:", cid)
}
