package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"kredly/internal/blockchain"
	"kredly/internal/database"
	"kredly/internal/models"
	"kredly/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)


type BlockchainHandler struct {
	client          *blockchain.CertificateClient
	metadataService *service.CertificateMetadataService
}

func NewBlockchainHandler(db *mongo.Database) *BlockchainHandler {
	blockchainService, err := blockchain.NewBlockchainService()
	if err != nil {
		return &BlockchainHandler{client: nil, metadataService: nil}
	}
	return &BlockchainHandler{
		client:          blockchainService.GetClient(),
		metadataService: service.NewCertificateMetadataService(db),
	}
}

// HandleGetCertificateMetadataByCertID retrieves certificate metadata by certificate ID
func (h *BlockchainHandler) HandleGetCertificateMetadataByCertID(c *gin.Context) {
	certificateID := c.Param("certificateId")

	if h.metadataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Metadata service not available"})
		return
	}

	metadata, err := h.metadataService.GetByCertificateID(certificateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if metadata == nil {
		// Metadata not found
		c.JSON(http.StatusOK, gin.H{
			"exists": false,
		})
		return
	}

	// Metadata found
	c.JSON(http.StatusOK, gin.H{
		"exists":   true,
		"metadata": metadata,
	})
}

// HandleGetCertificateMetadata retrieves certificate metadata by session ID
func (h *BlockchainHandler) HandleGetCertificateMetadata(c *gin.Context) {
	sessionID := c.Param("sessionId")

	if h.metadataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Metadata service not available"})
		return
	}

	metadata, err := h.metadataService.GetBySessionID(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if metadata == nil {
		// Metadata not found - certificate not yet issued
		c.JSON(http.StatusOK, gin.H{
			"exists": false,
		})
		return
	}

	// Metadata found - return it
	c.JSON(http.StatusOK, gin.H{
		"exists":   true,
		"metadata": metadata,
	})
}

// HandleGetUserCertificates retrieves all certificates for the authenticated user
func (h *BlockchainHandler) HandleGetUserCertificates(c *gin.Context) {
	if h.metadataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Metadata service not available"})
		return
	}

	var userID string
	if userInterface, exists := c.Get("user"); exists {
		if userMap, ok := userInterface.(gin.H); ok {
			if id, ok := userMap["id"].(string); ok {
				userID = id
			}
		}
	}

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	certificates, err := h.metadataService.GetByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"certificates": certificates,
	})
}

// Request/Response structs for verification
type VerifyResponse struct {
	IsValid  bool                        `json:"isValid"`
	Status   string                      `json:"status"`
	Message  string                      `json:"message"`
	Metadata *models.CertificateMetadata `json:"metadata,omitempty"`
}

// VerifyByCertificateIdRequest for verifying with certificate ID only
type VerifyByCertificateIdRequest struct {
	CertificateId string `json:"certificateId" binding:"required"`
}

// VerifyByHashOnlyRequest for verifying with only hash (no certificate ID needed)
type VerifyByHashOnlyRequest struct {
	PdfHash string `json:"pdfHash" binding:"required"`
}

// HandleVerifyByHashOnly verifies certificate by searching database with hash only
func (h *BlockchainHandler) HandleVerifyByHashOnly(c *gin.Context) {
	if h.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Blockchain service not available"})
		return
	}

	if h.metadataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Metadata service not available"})
		return
	}

	var req VerifyByHashOnlyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Search database by PDF hash to find certificate
	metadata, err := h.metadataService.GetByPdfHash(req.PdfHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if metadata == nil {
		// Certificate not found in database
		fmt.Printf("[Verify] Certificate not found in DB for hash: %s\n", req.PdfHash)
		c.JSON(http.StatusOK, VerifyResponse{
			IsValid: false,
			Status:  "NotFound",
			Message: "Certificate not found in database",
		})
		return
	}

	fmt.Printf("[Verify] Certificate found in DB:\n")
	fmt.Printf("  - Certificate ID: %s\n", metadata.CertificateID)
	fmt.Printf("  - Hash from upload: %s\n", req.PdfHash)
	fmt.Printf("  - Hash in database: %s\n", metadata.PdfHash)
	fmt.Printf("  - Match: %v\n", strings.EqualFold(req.PdfHash, metadata.PdfHash))

	// Found certificate in database, now verify with blockchain
	result, err := h.client.VerifyCertificate(metadata.CertificateID, req.PdfHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var message string
	switch result.Status {
	case blockchain.VerifyStatusValid:
		message = "Certificate is valid and authentic"
	case blockchain.VerifyStatusNotFound:
		message = "Certificate not found on blockchain"
	case blockchain.VerifyStatusRevoked:
		message = "Certificate has been revoked"
	case blockchain.VerifyStatusHashMismatch:
		message = "Certificate hash mismatch - document may be tampered"
	default:
		message = "Unknown status"
	}

	c.JSON(http.StatusOK, VerifyResponse{
		IsValid:  result.IsValid,
		Status:   result.Status.String(),
		Message:  message,
		Metadata: metadata,
	})
}

// HandleVerifyByCertificateId verifies certificate by searching database with certificate ID
func (h *BlockchainHandler) HandleVerifyByCertificateId(c *gin.Context) {
	if h.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Blockchain service not available"})
		return
	}

	if h.metadataService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Metadata service not available"})
		return
	}

	var req VerifyByCertificateIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Search database by certificate ID
	metadata, err := h.metadataService.GetByCertificateID(req.CertificateId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if metadata == nil {
		c.JSON(http.StatusOK, VerifyResponse{
			IsValid: false,
			Status:  "NotFound",
			Message: "Certificate not found in database",
		})
		return
	}

	// Found certificate in database, now verify with blockchain
	result, err := h.client.VerifyCertificate(metadata.CertificateID, metadata.PdfHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var message string
	switch result.Status {
	case blockchain.VerifyStatusValid:
		message = "Certificate is valid and authentic"
	case blockchain.VerifyStatusNotFound:
		message = "Certificate not found on blockchain"
	case blockchain.VerifyStatusRevoked:
		message = "Certificate has been revoked"
	case blockchain.VerifyStatusHashMismatch:
		message = "Certificate hash mismatch - document may be tampered"
	default:
		message = "Unknown status"
	}

	c.JSON(http.StatusOK, VerifyResponse{
		IsValid:  result.IsValid,
		Status:   result.Status.String(),
		Message:  message,
		Metadata: metadata,
	})
}

// HandleCheckCertificate checks if certificate exists (GET with query param)
func (h *BlockchainHandler) HandleCheckCertificate(c *gin.Context) {
	if h.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Blockchain service not available"})
		return
	}

	certificateID := c.Query("certificateId")
	if certificateID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "certificateId query parameter is required"})
		return
	}

	// Get certificate from blockchain
	cert, err := h.client.GetCertificate(certificateID)
	if err != nil {
		// Certificate not found
		c.JSON(http.StatusOK, gin.H{
			"isValid": false,
			"status":  "not_found",
			"message": "Certificate not found on blockchain",
		})
		return
	}

	// Check if revoked
	if cert.IsRevoked {
		c.JSON(http.StatusOK, gin.H{
			"isValid": false,
			"status":  "revoked",
			"message": "Certificate has been revoked",
		})
		return
	}

	// Certificate exists and valid
	c.JSON(http.StatusOK, gin.H{
		"isValid": true,
		"status":  "valid",
		"message": "Certificate exists and is valid",
	})
}

// IssueCertificateRequest for issuing new certificate
type IssueCertificateRequest struct {
	CertificateID  string `json:"certificateId" binding:"required"`
	SessionID      string `json:"sessionId" binding:"required"`
	PdfBuffer      string `json:"pdfBuffer" binding:"required"` // Base64 encoded PDF
	PdfHash        string `json:"pdfHash" binding:"required"`   // SHA256 hash from frontend
	RecipientName  string `json:"recipientName"`
	AssessmentName string `json:"assessmentName"`
	Score          int    `json:"score"`
}

// HandleIssueCertificate issues a new certificate to blockchain
func (h *BlockchainHandler) HandleIssueCertificate(c *gin.Context) {
	if h.client == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Blockchain service not available"})
		return
	}

	var req IssueCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID string
	if userInterface, exists := c.Get("user"); exists {
		if userMap, ok := userInterface.(gin.H); ok {
			if id, ok := userMap["id"].(string); ok {
				userID = id
			}
		}
	}

	// Decode base64 PDF buffer
	pdfBytes, err := base64.StdEncoding.DecodeString(req.PdfBuffer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PDF buffer: " + err.Error()})
		return
	}

	// Upload PDF to Pinata IPFS
	jwt := os.Getenv("PINATA_JWT")
	fmt.Printf("[DEBUG] HandleIssueCertificate - Loaded PINATA_JWT length: %d\n", len(jwt))
	if len(jwt) > 30 {
		fmt.Printf("[DEBUG] HandleIssueCertificate - Loaded PINATA_JWT prefix: %s\n", jwt[:30])
	}
	pinataClient := blockchain.NewPinataClient()
	if pinataClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Pinata service not available - check PINATA_JWT"})
		return
	}

	filename := fmt.Sprintf("Kredly_Certificate_%s.pdf", req.CertificateID)
	ipfsCID, err := pinataClient.UploadPDF(pdfBytes, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to IPFS: " + err.Error()})
		return
	}

	// Use hash from frontend (already calculated consistently)
	pdfHash := req.PdfHash
	fmt.Printf("[Issue] Using frontend-calculated hash: %s\n", pdfHash)

	// Issue certificate to blockchain
	txHash, err := h.client.IssueCertificate(req.CertificateID, pdfHash, ipfsCID)
	if err != nil {
		// Check if certificate already exists
		if strings.Contains(err.Error(), "CertificateAlreadyExists") {
			fmt.Printf("[Issue] Certificate already exists on blockchain, saving to DB...\n")

			// Save metadata to database even if certificate exists on blockchain
			// This prevents re-trying on every reload
			if h.metadataService != nil {
				metadata := &models.CertificateMetadata{
					SessionID:      req.SessionID,
					CertificateID:  req.CertificateID,
					UserID:         userID,
					RecipientName:  req.RecipientName,
					AssessmentName: req.AssessmentName,
					Score:          req.Score,
					PdfHash:        pdfHash,
					IpfsCID:        ipfsCID,
					IpfsURL:        blockchain.GetIPFSUrl(ipfsCID),
					TxHash:         "", // Empty since certificate already existed
				}

				if err := h.metadataService.Save(metadata); err != nil {
					fmt.Printf("Warning: Failed to save certificate metadata: %v\n", err)
				} else {
					fmt.Printf("[Issue] Metadata saved to DB for existing certificate\n")
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"success":       true,
				"certificateId": req.CertificateID,
				"pdfHash":       pdfHash,
				"ipfsCID":       ipfsCID,
				"ipfsUrl":       blockchain.GetIPFSUrl(ipfsCID),
				"txHash":        "",
				"message":       "Certificate already exists on blockchain",
				"alreadyExists": true,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save metadata to database for future reference
	if h.metadataService != nil {
		metadata := &models.CertificateMetadata{
			SessionID:      req.SessionID,
			CertificateID:  req.CertificateID,
			UserID:         userID,
			RecipientName:  req.RecipientName,
			AssessmentName: req.AssessmentName,
			Score:          req.Score,
			PdfHash:        pdfHash,
			IpfsCID:        ipfsCID,
			IpfsURL:        blockchain.GetIPFSUrl(ipfsCID),
			TxHash:         txHash,
		}

		if err := h.metadataService.Save(metadata); err != nil {
			// Log error but don't fail request (metadata save is optional)
			fmt.Printf("Warning: Failed to save certificate metadata: %v\n", err)
		}
	}

	// Log blockchain issued activity (get userID from context)
	if userInterface, exists := c.Get("user"); exists {
		if userMap, ok := userInterface.(gin.H); ok {
			if userID, ok := userMap["id"].(string); ok {
				models.LogActivityAsync(
					database.DB,
					userID,
					models.ActivityBlockchainIssued,
					fmt.Sprintf("Sertifikat %s Diterbitkan ke Blockchain", req.AssessmentName),
					fmt.Sprintf("Sertifikat Anda untuk %s telah diterbitkan ke blockchain dan tersimpan di IPFS", req.AssessmentName),
					&models.ActivityMetadata{
						TxHash: &txHash,
						Score:  &req.Score,
					},
				)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"certificateId": req.CertificateID,
		"pdfHash":       pdfHash,
		"ipfsCID":       ipfsCID,
		"ipfsUrl":       blockchain.GetIPFSUrl(ipfsCID),
		"txHash":        txHash,
		"message":       "Certificate issued successfully",
	})
}
