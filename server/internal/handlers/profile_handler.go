package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"kredly/internal/blockchain"
	"kredly/internal/database"
	"kredly/internal/groq"
	"kredly/internal/models"
	"kredly/internal/pdf"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProfileHandler struct {
	groqClient *groq.Client
}

func NewProfileHandler(groqClient *groq.Client) *ProfileHandler {
	return &ProfileHandler{
		groqClient: groqClient,
	}
}

// HandleCheckUsername checks if a username is already taken
func (h *ProfileHandler) HandleCheckUsername(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	userColl := database.DB.Collection("user")
	var existingUser models.User
	err := userColl.FindOne(c.Request.Context(), bson.M{"username": username}).Decode(&existingUser)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username sudah digunakan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"available": true})
}

// HandleGetProfile returns the UserProfile for the authenticated user
func (h *ProfileHandler) HandleGetProfile(c *gin.Context) {
	// Ambil user dari context (sudah di-set oleh middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// User dari middleware adalah gin.H (map), extract userID
	userMap, ok := userInterface.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	// Debug log
	println("🔍 [DEBUG] Fetching UserProfile for userId:", userID)

	// Fetch UserProfile dari database
	userProfileColl := database.DB.Collection("userProfile")
	var userProfile models.UserProfile
	err := userProfileColl.FindOne(c.Request.Context(), bson.M{"userId": userID}).Decode(&userProfile)

	if err != nil {
		// Debug log
		println("❌ [DEBUG] UserProfile not found:", err.Error())

		// UserProfile tidak ditemukan
		c.JSON(http.StatusOK, gin.H{
			"profile": nil,
			"message": "No profile found",
			"userId":  userID, // Return userId for debugging
		})
		return
	}

	// Enrich in-progress assessments with the latest ExpiresAt from cat_sessions
	catSessionsColl := database.DB.Collection("cat_sessions")
	for i, ast := range userProfile.CVAssessments {
		if ast.Status == "in-progress" && ast.SessionID != "" {
			var sess models.Session
			err := catSessionsColl.FindOne(c.Request.Context(), bson.M{"_id": ast.SessionID}).Decode(&sess)
			if err == nil {
				userProfile.CVAssessments[i].ExpiresAt = &sess.ExpiresAt
			}
		}
	}

	// Debug log
	println("✅ [DEBUG] UserProfile found:", userProfile.ID)

	c.JSON(http.StatusOK, gin.H{
		"profile": userProfile,
	})
}

// HandleUpdateProfile updates user's name and username
func (h *ProfileHandler) HandleUpdateProfile(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userMap, ok := userInterface.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Name     string `json:"name" binding:"required"`
		Username string `json:"username"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	ctx := c.Request.Context()
	userColl := database.DB.Collection("user")

	// Check if username is being changed and if it's unique
	if req.Username != "" {
		var existingUser models.User
		err := userColl.FindOne(ctx, bson.M{
			"username": req.Username,
			"_id":      bson.M{"$ne": userID},
		}).Decode(&existingUser)

		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Username sudah digunakan"})
			return
		}
	}

	// Build update document
	updateDoc := bson.M{
		"name":      req.Name,
		"updatedAt": time.Now(),
	}

	if req.Username != "" {
		updateDoc["username"] = req.Username
	}

	// Update user
	result, err := userColl.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": updateDoc},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Profil berhasil diperbarui",
		"name":     req.Name,
		"username": req.Username,
	})
}

// HandleUploadCV handles CV file upload, parsing, and updating UserProfile
func (h *ProfileHandler) HandleUploadCV(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userMap, ok := userInterface.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("cv")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File CV tidak ditemukan"})
		return
	}

	// Validate file type
	if file.Header.Get("Content-Type") != "application/pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File harus berformat PDF"})
		return
	}

	// Validate file size (max 5MB)
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran file maksimal 5MB"})
		return
	}

	// Create uploads directory if not exists
	uploadDir := "./uploads/cv"
	if os.Getenv("VERCEL") == "1" {
		uploadDir = "/tmp/uploads/cv"
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat direktori upload"})
		return
	}

	// Generate unique filename
	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%s_%s%s", userID, uuid.New().String(), fileExt)
	filePath := filepath.Join(uploadDir, fileName)

	// Save the file permanently
	if err := saveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file"})
		return
	}
	ctx := c.Request.Context()
	userProfileColl := database.DB.Collection("userProfile")

	// Get existing profile to see if there's existing CV data to update/merge
	var existingProfile models.UserProfile
	hasExistingProfile := false
	if err := userProfileColl.FindOne(ctx, bson.M{"userId": userID}).Decode(&existingProfile); err == nil {
		hasExistingProfile = true
	}

	// Parse CV - extract text and analyze with Groq
	var parsedRole, parsedLevel, parsedSummary string
	var parsedSkills []string
	var parsedAssessments []models.GeneratedAssessment

	// This parsing is optional - if it fails, we still save the file
	if text, err := pdf.ReadPDFText(filePath); err == nil && len(text) > 0 {
		var existingDataStr string
		if hasExistingProfile {
			var role, level, summary string
			var skills []string
			if existingProfile.CVRole != nil {
				role = *existingProfile.CVRole
			}
			if existingProfile.CVLevel != nil {
				level = *existingProfile.CVLevel
			}
			if existingProfile.CVSummary != nil {
				summary = *existingProfile.CVSummary
			}
			skills = existingProfile.CVSkills

			existingDataBytes, _ := json.Marshal(map[string]interface{}{
				"role":        role,
				"level":       level,
				"skills":      skills,
				"summary":     summary,
				"assessments": existingProfile.CVAssessments,
			})
			existingDataStr = string(existingDataBytes)
		}

		// Use improved system prompt without programming bias
		systemPrompt := `You are an expert ATS (Applicant Tracking System) CV parser.
Your PRIMARY TASK is to accurately extract information ONLY from the CV text provided. Do NOT make assumptions or add skills that are not explicitly mentioned in the CV.

CRITICAL RULES:
1. Extract ONLY skills that are explicitly written in the CV text
2. Identify the candidate's actual field/domain from their education, experience, and listed skills
3. Do NOT assume programming/software development unless clearly stated in the CV
4. Match the role and category to the candidate's actual domain (e.g., Data Analyst, Mathematician, Marketing, Finance, etc.)
`

		if existingDataStr != "" {
			systemPrompt += fmt.Sprintf(`Here is the user's existing parsed CV profile data and assessments:
%s

Please review the new CV text, merge/update the fields accordingly while preserving completed/in-progress assessments.
`, existingDataStr)
		}

		systemPrompt += `Extract all information and output a structured JSON object following this exact schema:
{
  "role": "Candidate's PRIMARY role based on their education, experience, and skills (e.g., Data Analyst, Software Engineer, Marketing Manager, Financial Analyst, Mathematician)",
  "level": "Candidate's seniority level based on experience (e.g., Entry Level, Junior, Mid, Senior, Lead)",
  "skills": ["ONLY skills explicitly mentioned in the CV - do NOT add programming skills if not present"],
  "summary": "A concise summary of the candidate's ACTUAL profile based on CV content. Keep it brief (2-3 sentences max) as plain paragraph.",
  "assessments": [
    {
      "id": "unique-slug-based-on-actual-domain",
      "type": "general", "skill", or "related_skill",
      "title": "Assessment title matching candidate's ACTUAL domain",
      "description": "What is tested in this assessment",
      "difficulty": "Beginner", "Intermediate", or "Advanced" based on candidate level,
      "estimatedTime": "90 menit for general, 45 menit for skill/related_skill",
      "questionCount": 50 for general, 30 for skill/related_skill,
      "topics": ["relevant topics from candidate's domain"] (only for "general" type),
      "isRecommended": true,
      "category": "Match candidate's actual domain (e.g., Data Analysis, Mathematics, Marketing, Finance, Software Development, Design, etc.)",
      "status": "available"
    }
  ]
}

ASSESSMENT GENERATION RULES:
1. Generate 1 "general" assessment matching the candidate's ACTUAL overall domain. questionCount=50, estimatedTime="90 menit"
2. Generate 3-5 skill assessments based on ACTUAL skills from the CV:
   - 2-3 assessments matching their TOP extracted skills (type="skill")
   - 1-2 complementary skills within their ACTUAL domain (type="related_skill")
   - questionCount=30, estimatedTime="45 menit"
3. If existing data has completed/in-progress assessments, PRESERVE them in the output
4. Use appropriate category based on ACTUAL domain

Return ONLY valid JSON object without markdown tags.`

		groqReq := groq.ChatCompletionRequest{
			Messages: []groq.Message{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: text},
			},
			Model: "qwen/qwen3-32b",
			ResponseFormat: &groq.ResponseFormat{
				Type: "json_object",
			},
		}

		resp, err := h.groqClient.CreateChatCompletion(groqReq)
		if err != nil {
			println("❌ [DEBUG] Groq CreateChatCompletion error:", err.Error())
		} else if len(resp.Choices) == 0 {
			println("❌ [DEBUG] Groq returned 0 choices")
		} else {
			var parsed struct {
				Role        string                       `json:"role"`
				Level       string                       `json:"level"`
				Skills      []string                     `json:"skills"`
				Summary     string                       `json:"summary"`
				Assessments []models.GeneratedAssessment `json:"assessments"`
			}
			unmarshalErr := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &parsed)
			if unmarshalErr != nil {
				println("❌ [DEBUG] Groq JSON unmarshal error:", unmarshalErr.Error(), "content:", resp.Choices[0].Message.Content)
			} else {
				parsedRole = parsed.Role
				parsedLevel = parsed.Level
				parsedSkills = parsed.Skills
				parsedSummary = parsed.Summary
				parsedAssessments = parsed.Assessments
				println("✅ [DEBUG] Groq successfully parsed CV data - Role:", parsedRole, "Level:", parsedLevel, "Assessments:", len(parsedAssessments))
			}
		}
	} else {
		if err != nil {
			println("❌ [DEBUG] ReadPDFText error:", err.Error())
		} else {
			println("❌ [DEBUG] ReadPDFText returned empty text")
		}
	}

	// Prepare update document
	updateDoc := bson.M{
		"cvFileName": file.Filename,
		"cvFilePath": filePath,
		"updatedAt":  time.Now(),
	}

	// Add parsed data if available
	if parsedRole != "" {
		updateDoc["cvRole"] = parsedRole
	}
	if parsedLevel != "" {
		updateDoc["cvLevel"] = parsedLevel
	}
	if len(parsedSkills) > 0 {
		updateDoc["cvSkills"] = parsedSkills
	}
	if parsedSummary != "" {
		updateDoc["cvSummary"] = parsedSummary
	}
	if len(parsedAssessments) > 0 {
		updateDoc["cvAssessments"] = parsedAssessments
	}

	// Update or insert UserProfile with upsert
	filter := bson.M{"userId": userID}
	update := bson.M{"$set": updateDoc}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var updatedProfile models.UserProfile
	err = userProfileColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedProfile)

	// If still error after upsert, try manual insert
	if err != nil {
		newProfile := models.UserProfile{
			ID:         uuid.New().String(),
			UserID:     userID,
			CVFileName: file.Filename,
			CVFilePath: filePath,
			Experience: "not-specified",
			IsStudent:  false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if parsedRole != "" {
			newProfile.CVRole = &parsedRole
		}
		if parsedLevel != "" {
			newProfile.CVLevel = &parsedLevel
		}
		if len(parsedSkills) > 0 {
			newProfile.CVSkills = parsedSkills
		}
		if parsedSummary != "" {
			newProfile.CVSummary = &parsedSummary
		}
		if len(parsedAssessments) > 0 {
			newProfile.CVAssessments = parsedAssessments
		}

		_, err = userProfileColl.InsertOne(ctx, newProfile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan profil"})
			return
		}
		updatedProfile = newProfile
	}

	// Return response with assessments
	response := gin.H{
		"message":     "CV berhasil diupload dan diparse",
		"cvFileName":  file.Filename,
		"cvFilePath":  filePath,
		"role":        parsedRole,
		"level":       parsedLevel,
		"skills":      parsedSkills,
		"summary":     parsedSummary,
		"assessments": parsedAssessments,
	}

	c.JSON(http.StatusOK, response)
}

// HandleDeleteAccount permanently deletes a user account and all related data
func (h *ProfileHandler) HandleDeleteAccount(c *gin.Context) {
	// Get user from context (set by middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Extract user data
	userMap, ok := userInterface.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	userEmail, ok := userMap["email"].(string)
	if !ok || userEmail == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user email"})
		return
	}

	ctx := c.Request.Context()

	// Start deleting all related data
	// 1. Delete all Sessions
	sessionColl := database.DB.Collection("session")
	_, err := sessionColl.DeleteMany(ctx, bson.M{"userId": userID})
	if err != nil {
		println("⚠️ [DELETE] Failed to delete sessions:", err.Error())
	}

	// 2. Delete all Accounts (OAuth connections)
	accountColl := database.DB.Collection("account")
	_, err = accountColl.DeleteMany(ctx, bson.M{"userId": userID})
	if err != nil {
		println("⚠️ [DELETE] Failed to delete accounts:", err.Error())
	}

	// 3. Delete UserProfile
	userProfileColl := database.DB.Collection("userProfile")
	_, err = userProfileColl.DeleteOne(ctx, bson.M{"userId": userID})
	if err != nil {
		println("⚠️ [DELETE] Failed to delete user profile:", err.Error())
	}

	// 4. Delete Verifications (email verification tokens)
	verificationColl := database.DB.Collection("verification")
	_, err = verificationColl.DeleteMany(ctx, bson.M{"identifier": userEmail})
	if err != nil {
		println("⚠️ [DELETE] Failed to delete verifications:", err.Error())
	}

	// 5. Finally, delete the User document
	userColl := database.DB.Collection("user")
	result, err := userColl.DeleteOne(ctx, bson.M{"_id": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user account"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Clear the session cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)

	println("✅ [DELETE] Successfully deleted user account:", userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Account successfully deleted",
	})
}

// HandleGetPublicProfileSettings returns public profile settings for authenticated user
func (h *ProfileHandler) HandleGetPublicProfileSettings(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userMap, ok := userInterface.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx := c.Request.Context()
	settingsColl := database.DB.Collection("publicProfileSettings")

	var settings models.PublicProfileSettings
	err := settingsColl.FindOne(ctx, bson.M{"userId": userID}).Decode(&settings)
	if err != nil {
		// Return default settings if not found
		defaultSettings := models.PublicProfileSettings{
			ID:               uuid.New().String(),
			UserID:           userID,
			IsPublic:         true,
			Headline:         "",
			Bio:              "",
			ShowCertificates: true,
			ShowAssessments:  true,
			ShowSkills:       true,
			ShowCVData:       false,
			SocialLinks: models.SocialLinks{
				Linkedin:  "",
				Github:    "",
				Portfolio: "",
				Twitter:   "",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		c.JSON(http.StatusOK, gin.H{
			"settings": defaultSettings,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
	})
}

// HandleUpdatePublicProfileSettings updates public profile settings for authenticated user
func (h *ProfileHandler) HandleUpdatePublicProfileSettings(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userMap, ok := userInterface.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		IsPublic         bool               `json:"isPublic"`
		Headline         string             `json:"headline"`
		Bio              string             `json:"bio"`
		ShowCertificates bool               `json:"showCertificates"`
		ShowAssessments  bool               `json:"showAssessments"`
		ShowSkills       bool               `json:"showSkills"`
		ShowCVData       bool               `json:"showCVData"`
		SocialLinks      models.SocialLinks `json:"socialLinks"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	ctx := c.Request.Context()
	settingsColl := database.DB.Collection("publicProfileSettings")

	filter := bson.M{"userId": userID}
	update := bson.M{
		"$set": bson.M{
			"isPublic":         req.IsPublic,
			"headline":         req.Headline,
			"bio":              req.Bio,
			"showCertificates": req.ShowCertificates,
			"showAssessments":  req.ShowAssessments,
			"showSkills":       req.ShowSkills,
			"showCVData":       req.ShowCVData,
			"socialLinks":      req.SocialLinks,
			"updatedAt":        time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := settingsColl.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pengaturan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pengaturan profil publik berhasil disimpan",
	})
}

// HandleGetPublicProfileByUsername returns public profile data by username
func (h *ProfileHandler) HandleGetPublicProfileByUsername(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	ctx := c.Request.Context()
	userColl := database.DB.Collection("user")

	var user models.User
	err := userColl.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	userProfileColl := database.DB.Collection("userProfile")
	var userProfile models.UserProfile
	_ = userProfileColl.FindOne(ctx, bson.M{"userId": user.ID}).Decode(&userProfile)

	settingsColl := database.DB.Collection("publicProfileSettings")
	var settings models.PublicProfileSettings
	err = settingsColl.FindOne(ctx, bson.M{"userId": user.ID}).Decode(&settings)
	if err != nil {
		settings = models.PublicProfileSettings{
			ShowCertificates: true,
			ShowAssessments:  true,
			ShowSkills:       true,
			ShowCVData:       false,
			IsPublic:         true,
		}
	}

	// Check if profile is public
	if !settings.IsPublic {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profil tidak ditemukan atau tidak publik"})
		return
	}

	var sessionIDs []string
	sessionColl := database.DB.Collection("session")
	cursor, err := sessionColl.Find(ctx, bson.M{"userId": user.ID, "completed": true})
	if err == nil {
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var sess models.Session
			if err := cursor.Decode(&sess); err == nil {
				sessionIDs = append(sessionIDs, sess.ID)
			}
		}
	}
	log.Printf("[DEBUG] User %s (%s) has %d completed sessions", user.Name, user.ID, len(sessionIDs))

	certs := []map[string]interface{}{}
	if settings.ShowCertificates {
		certColl := database.DB.Collection("certificate_metadata")
		var filter bson.M
		if len(sessionIDs) > 0 {
			filter = bson.M{
				"$or": []bson.M{
					{"user_id": user.ID},
					{"session_id": bson.M{"$in": sessionIDs}},
				},
			}
		} else {
			filter = bson.M{"user_id": user.ID}
		}

		certCursor, err := certColl.Find(ctx, filter)
		if err == nil {
			defer certCursor.Close(ctx)
			for certCursor.Next(ctx) {
				var cert models.CertificateMetadata
				if err := certCursor.Decode(&cert); err == nil {
					certs = append(certs, map[string]interface{}{
						"id":              cert.CertificateID,
						"sessionId":       cert.SessionID,
						"title":           cert.AssessmentName,
						"score":           cert.Score,
						"issuedAt":        cert.CreatedAt.Format(time.RFC3339),
						"verificationUrl": cert.IpfsURL,
					})
				}
			}
		} else {
			log.Printf("[DEBUG] Error fetching certificates: %v", err)
		}
	}
	log.Printf("[DEBUG] ShowCertificates=%v, SessionIDs=%d, Certificates=%d", settings.ShowCertificates, len(sessionIDs), len(certs))

	var completedAssessments []map[string]interface{}
	if settings.ShowAssessments {
		cursor, err := sessionColl.Find(ctx, bson.M{
			"userId":    user.ID,
			"completed": true,
			"result":    bson.M{"$exists": true},
		})
		if err == nil {
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var sess models.Session
				if err := cursor.Decode(&sess); err == nil {
					completedAt := sess.CreatedAt
					if sess.CompletedAt != nil {
						completedAt = *sess.CompletedAt
					}
					score := 0
					if sess.Result != nil {
						score = sess.Result.Score
					}
					completedAssessments = append(completedAssessments, map[string]interface{}{
						"id":          sess.ID,
						"title":       fmt.Sprintf("%s Assessment (%s)", sess.Role, sess.Level),
						"score":       score,
						"completedAt": completedAt.Format(time.RFC3339),
						"status":      "completed",
					})
				}
			}
		}
	}

	var cvSkills []string
	if settings.ShowSkills {
		cvSkills = userProfile.CVSkills
	}

	var imageVal string
	if user.Image != nil {
		imageVal = *user.Image
	}

	profileResponse := gin.H{
		"id":           user.ID,
		"name":         user.Name,
		"username":     username,
		"image":        imageVal,
		"headline":     settings.Headline,
		"bio":          settings.Bio,
		"cvRole":       userProfile.CVRole,
		"cvLevel":      userProfile.CVLevel,
		"cvSkills":     cvSkills,
		"certificates": certs,
		"assessments":  completedAssessments,
		"socialLinks":  settings.SocialLinks,
		"displaySettings": gin.H{
			"showCertificates": settings.ShowCertificates,
			"showAssessments":  settings.ShowAssessments,
			"showSkills":       settings.ShowSkills,
			"showCVData":       settings.ShowCVData,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"profile": profileResponse,
	})
}

// HandleUploadProfilePhoto handles profile photo upload to Pinata IPFS
func (h *ProfileHandler) HandleUploadProfilePhoto(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userMap, ok := userInterface.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File foto tidak ditemukan"})
		return
	}

	// Validate file type (only images)
	contentType := file.Header.Get("Content-Type")
	allowedTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	isValidType := false
	for _, t := range allowedTypes {
		if contentType == t {
			isValidType = true
			break
		}
	}
	if !isValidType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File harus berformat JPG, PNG, GIF, atau WebP"})
		return
	}

	// Validate file size (max 2MB)
	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran file maksimal 2MB"})
		return
	}

	// Read file buffer
	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file"})
		return
	}
	defer openedFile.Close()

	imageBuffer, err := io.ReadAll(openedFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file"})
		return
	}

	ctx := c.Request.Context()
	userColl := database.DB.Collection("user")

	// Get current user to check for old image
	var currentUser models.User
	err = userColl.FindOne(ctx, bson.M{"_id": userID}).Decode(&currentUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// Upload to Pinata
	pinataClient := blockchain.NewPinataClient()
	if pinataClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Pinata service not available - check PINATA_JWT"})
		return
	}

	// Generate filename
	filename := fmt.Sprintf("profile_%s_%d%s", userID, time.Now().Unix(), filepath.Ext(file.Filename))
	ipfsCID, err := pinataClient.UploadImage(imageBuffer, filename, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto: " + err.Error()})
		return
	}

	// Generate IPFS gateway URL
	imageURL := blockchain.GetIPFSUrl(ipfsCID)

	// Delete old image from Pinata if exists
	if currentUser.Image != nil && *currentUser.Image != "" {
		oldCID := blockchain.ExtractCIDFromURL(*currentUser.Image)
		if oldCID != "" {
			go func() {
				if err := pinataClient.DeleteFile(oldCID); err != nil {
					println("⚠️ [DEBUG] Failed to delete old profile photo from IPFS:", err.Error())
				} else {
					println("✅ [DEBUG] Old profile photo deleted from IPFS:", oldCID)
				}
			}()
		}
	}

	// Update user.image in database
	_, err = userColl.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{
			"$set": bson.M{
				"image":     imageURL,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan URL foto"})
		return
	}

	println("✅ [DEBUG] Profile photo uploaded successfully - CID:", ipfsCID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Foto profil berhasil diupload",
		"imageUrl": imageURL,
		"ipfsCID":  ipfsCID,
	})
}
