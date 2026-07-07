package handlers

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"kredly/internal/database"
	"kredly/internal/groq"
	"kredly/internal/models"
	"kredly/internal/pdf"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CVHandler struct {
	groqClient *groq.Client
}

func NewCVHandler(groqClient *groq.Client) *CVHandler {
	return &CVHandler{
		groqClient: groqClient,
	}
}

// HandleParseCV receives a PDF file, extracts text, sends it to Groq API, and returns JSON
func (h *CVHandler) HandleParseCV(c *gin.Context) {
	// Get logged-in user ID from context (if authenticated)
	var loggedInUserID string
	if userVal, exists := c.Get("user"); exists {
		if userMap, ok := userVal.(gin.H); ok {
			if uid, ok := userMap["id"].(string); ok {
				loggedInUserID = uid
			}
		}
	}

	// Fetch existing profile to see if there's existing CV data and assessments
	var existingProfile models.UserProfile
	hasExistingProfile := false
	if loggedInUserID != "" {
		userProfileColl := database.DB.Collection("userProfile")
		if err := userProfileColl.FindOne(c.Request.Context(), bson.M{"userId": loggedInUserID}).Decode(&existingProfile); err == nil {
			hasExistingProfile = true
		}
	}

	// Validasi saldo token jika ini merupakan upload ulang (profil sudah ada)
	if hasExistingProfile {
		userColl := database.DB.Collection("user")
		var dbUser models.User
		if err := userColl.FindOne(c.Request.Context(), bson.M{"_id": loggedInUserID}).Decode(&dbUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User tidak ditemukan"})
			return
		}

		if dbUser.TokenBalance == nil || dbUser.TokenBalance.Current < 3 {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Saldo kredit Anda tidak mencukupi untuk melakukan upload ulang CV. Silakan top up kredit terlebih dahulu (Dibutuhkan 3 token).",
			})
			return
		}
	}

	// 1. Get file from form
	file, err := c.FormFile("cv")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file from request: " + err.Error()})
		return
	}

	// 2. Create a temporary file
	tempFile, err := os.CreateTemp("", "cv-*.pdf")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp file: " + err.Error()})
		return
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	// 3. Save uploaded file to temp path
	if err := saveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}

	// 4. Extract text from PDF
	text, err := pdf.ReadPDFText(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract text from PDF: " + err.Error()})
		return
	}

	if len(text) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The uploaded PDF file does not contain any readable text"})
		return
	}

	// 5. Build prompt for Groq
	var systemPrompt string
	if hasExistingProfile {
		var role, level, summary string
		if existingProfile.CVRole != nil {
			role = *existingProfile.CVRole
		}
		if existingProfile.CVLevel != nil {
			level = *existingProfile.CVLevel
		}
		if existingProfile.CVSummary != nil {
			summary = *existingProfile.CVSummary
		}

		existingDataBytes, _ := json.Marshal(map[string]interface{}{
			"role":        role,
			"level":       level,
			"skills":      existingProfile.CVSkills,
			"summary":     summary,
			"assessments": existingProfile.CVAssessments,
		})
		existingDataStr := string(existingDataBytes)

		systemPrompt = `You are an expert ATS (Applicant Tracking System) CV parser. 
Your task is to update or merge a candidate's existing CV profile data and recommended assessments with the text extracted from their newly uploaded CV.

Here is the candidate's existing data and recommended assessments:
` + existingDataStr + `

Extract all information from the provided new CV text, merge it with the existing data, and output a structured JSON object.
The JSON must follow this exact schema:
{
  "role": "Candidate's primary role or title (e.g., Backend Engineer)",
  "level": "Candidate's seniority level (e.g., Junior, Mid, Senior, Lead)",
  "skills": ["Skill 1", "Skill 2"],
  "summary": "A concise, clean, and professional summary of the candidate's profile. Keep it brief (2-3 sentences max) as a single plain paragraph. Do NOT include any special characters, list bullet points, or symbols like '●' or '•'.",
  "assessments": [
    {
      "id": "A unique slug/ID like gen-frontend, skill-typescript, or rel-deep-learning",
      "type": "general", "skill", or "related_skill",
      "title": "Assessment title (e.g. Front End, Backend, TypeScript, Go)",
      "description": "Short description of what is tested",
      "difficulty": "Beginner", "Intermediate", or "Advanced" based on candidate level,
      "estimatedTime": "Estimated time (e.g. '90 menit' for general, '45 menit' for skill/related_skill)",
      "questionCount": number of questions (e.g., 50 for general, 30 for skill/related_skill),
      "topics": ["list of key topics/subjects tested"] (only for type "general", empty or null for other types),
      "isRecommended": true,
      "category": "Frontend", "Backend", "Mobile", "DevOps", "Database", "General" etc.,
      "status": "available" | "in-progress" | "completed",
      "progress": 0,
      "sessionId": "",
      "score": 0,
      "level": ""
    }
  ]
}

CRITICAL RULES FOR MERGING/UPDATING "assessments":
1. PRESERVE COMPLETED & IN-PROGRESS: Any assessment in the existing data with status "completed" or "in-progress" MUST be preserved EXACTLY as is. Do not modify its status, title, score, progress, id, sessionId, difficulty, or category. Include it in the final "assessments" list.
2. MERGE RECOMMENDED ASSESSMENTS:
   - Identify new skills or roles from the new CV.
   - Generate exactly 1 "general" assessment matching the candidate's updated overall role (e.g., "Front End", "Backend"). The "questionCount" must be 50 and "estimatedTime" must be "90 menit". If a general assessment already exists and is completed/in-progress, keep it; otherwise update it.
   - Generate 3-5 skill-related assessments matching their top skills: 2-3 matching top extracted skills ("type": "skill"), and 1-2 highly related/complementary ("type": "related_skill"). The "questionCount" for these must be 30 and "estimatedTime" must be "45 menit".
   - If any new recommended assessment is already in the existing assessments list, keep its existing structure (especially if it is completed or in-progress).
   - Add the new recommended assessments to the list. Avoid duplicate assessments (match by ID or Title).
3. DIFFICULTY: Choose the "difficulty" matching their cv level (Junior -> Beginner/Intermediate, Mid -> Intermediate, Senior/Lead -> Advanced).
4. Assign a unique slug/ID for each new assessment (e.g., "gen-frontend" for general, "skill-typescript" for skill, "rel-deep-learning" for related_skill).

Ensure all fields are populated as accurately as possible based on the text. If a field is missing, set it to null or an empty array/string. Do not include any markdown format tags like ` + "`" + `json` + "`" + ` or any conversational intro/outro text. Return ONLY the raw JSON object.`
	} else {
		systemPrompt = `You are an expert ATS (Applicant Tracking System) CV parser.
Your PRIMARY TASK is to accurately extract information ONLY from the CV text provided. Do NOT make assumptions or add skills that are not explicitly mentioned in the CV.

CRITICAL RULES:
1. Extract ONLY skills that are explicitly written in the CV text
2. Identify the candidate's actual field/domain from their education, experience, and listed skills
3. Do NOT assume programming/software development unless clearly stated in the CV
4. Match the role and category to the candidate's actual domain (e.g., Data Analyst, Mathematician, Marketing, Finance, etc.)

Extract all information and output a structured JSON object following this exact schema:
{
  "role": "Candidate's PRIMARY role based on their education, experience, and skills (e.g., Data Analyst, Software Engineer, Marketing Manager, Financial Analyst, Mathematician)",
  "level": "Candidate's seniority level based on experience (e.g., Entry Level, Junior, Mid, Senior, Lead)",
  "skills": ["ONLY skills explicitly mentioned in the CV - do NOT add programming skills if not present"],
  "summary": "A concise summary of the candidate's ACTUAL profile based on CV content. Keep it brief (2-3 sentences max) as plain paragraph. Do NOT include special characters or bullet points.",
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
1. Generate 1 "general" assessment matching the candidate's ACTUAL overall domain with 3-5 relevant topics. questionCount=50, estimatedTime="90 menit"
2. Generate 3-5 skill assessments based on ACTUAL skills from the CV:
   - 2-3 assessments matching their TOP extracted skills (type="skill")
   - 1-2 complementary skills within their ACTUAL domain (type="related_skill")
   - questionCount=30, estimatedTime="45 menit"
3. Set difficulty based on cv level: Entry/Junior -> Beginner/Intermediate, Mid -> Intermediate, Senior/Lead -> Advanced
4. Use appropriate category based on ACTUAL domain:
   - Data/Statistics: "Data Analysis", "Statistics", "Mathematics"
   - Software: "Frontend", "Backend", "Mobile", "DevOps"
   - Business: "Marketing", "Finance", "Management"
   - Design: "UI/UX", "Graphic Design"
   - etc.

EXAMPLE FOR NON-PROGRAMMING CV:
If CV shows: "S1 Matematika, skills: Analisis Data, Statistika, R, Excel"
Then generate: role="Data Analyst", skills=["Analisis Data", "Statistika", "R", "Excel"], category="Data Analysis"
NOT: role="Backend Engineer", skills=["Python", "JavaScript"], category="Backend"

Return ONLY valid JSON object without markdown tags or explanatory text.`
	}

	// 6. Request Chat Completion from Groq with JSON Mode
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to parse CV content via Groq API",
			"details": err.Error(),
		})
		return
	}

	if len(resp.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Groq API returned an empty completion"})
		return
	}

	rawContent := resp.Choices[0].Message.Content
	cleanJSON := strings.TrimSpace(rawContent)

	// Clean markdown block wrappers if present (e.g. ```json ... ```)
	if strings.HasPrefix(cleanJSON, "```") {
		lines := strings.Split(cleanJSON, "\n")
		var validLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "```") {
				validLines = append(validLines, line)
			}
		}
		cleanJSON = strings.Join(validLines, "\n")
	}
	cleanJSON = strings.TrimSpace(cleanJSON)

	// 7. Save parsed results to the user profile if logged in
	type parsedCV struct {
		Role        string                       `json:"role"`
		Level       string                       `json:"level"`
		Skills      []string                     `json:"skills"`
		Summary     string                       `json:"summary"`
		Assessments []models.GeneratedAssessment `json:"assessments"`
	}

	var parsed parsedCV
	if err := json.Unmarshal([]byte(cleanJSON), &parsed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Groq response: " + err.Error()})
		return
	}

	cvSummary := parsed.Summary
	if cvSummary == "" {
		cleanedText := text
		cleanedText = strings.ReplaceAll(cleanedText, "●", "")
		cleanedText = strings.ReplaceAll(cleanedText, "•", "")
		cleanedText = strings.ReplaceAll(cleanedText, "*", "")
		words := strings.Fields(cleanedText)
		cleanedText = strings.Join(words, " ")
		if len(cleanedText) > 400 {
			cleanedText = cleanedText[:397] + "..."
		}
		cvSummary = cleanedText
	}
	parsed.Summary = cvSummary

	// Merge assessments in Go to prevent LLM mistakes from resetting status
	if hasExistingProfile && len(existingProfile.CVAssessments) > 0 {
		var finalAssessments []models.GeneratedAssessment
		existingMap := make(map[string]models.GeneratedAssessment)
		existingTitleMap := make(map[string]models.GeneratedAssessment)

		// 1. First, preserve all completed and in-progress assessments
		for _, a := range existingProfile.CVAssessments {
			existingMap[a.ID] = a
			cleanTitle := strings.ToLower(strings.TrimSpace(a.Title))
			existingTitleMap[cleanTitle] = a

			if a.Status == "completed" || a.Status == "in-progress" {
				finalAssessments = append(finalAssessments, a)
			}
		}

		// 2. Next, process recommendations from the new parser output
		for _, newAst := range parsed.Assessments {
			cleanNewTitle := strings.ToLower(strings.TrimSpace(newAst.Title))

			var existingAst models.GeneratedAssessment
			found := false

			if est, ok := existingMap[newAst.ID]; ok {
				existingAst = est
				found = true
			} else if est, ok := existingTitleMap[cleanNewTitle]; ok {
				existingAst = est
				found = true
			}

			if found {
				// If the existing one is completed or in-progress, we already added it, so skip
				if existingAst.Status == "completed" || existingAst.Status == "in-progress" {
					continue
				}
				// Otherwise, keep the existing one to preserve its fields
				finalAssessments = append(finalAssessments, existingAst)
			} else {
				// Completely new recommendation. Ensure it starts as "available"
				newAst.Status = "available"
				finalAssessments = append(finalAssessments, newAst)
			}
		}

		parsed.Assessments = finalAssessments
	}

	if loggedInUserID != "" {
		userColl := database.DB.Collection("user")
		now := time.Now()

		userUpdateDoc := bson.M{
			"cvRole":     parsed.Role,
			"cvLevel":    parsed.Level,
			"cvSkills":   parsed.Skills,
			"cvSummary":  parsed.Summary,
			"cvParsedAt": now,
			"updatedAt":  now,
		}

		update := bson.M{
			"$set": userUpdateDoc,
		}

		// Jika ini adalah re-upload (profil sudah ada), kurangi token
		if hasExistingProfile {
			update["$inc"] = bson.M{
				"tokenBalance.current":    -3,
				"tokenBalance.totalSpent": 3,
			}
		}

		_, updateErr := userColl.UpdateOne(c.Request.Context(), bson.M{"_id": loggedInUserID}, update)
		if updateErr != nil {
			// Log the error but don't fail the request since parsing succeeded
			c.Error(updateErr)
		}

		// Update UserProfile document as well
		userProfileColl := database.DB.Collection("userProfile")
		profileUpdate := bson.M{
			"$set": bson.M{
				"cvRole":        parsed.Role,
				"cvLevel":       parsed.Level,
				"cvSkills":      parsed.Skills,
				"cvSummary":     parsed.Summary,
				"cvAssessments": parsed.Assessments,
				"updatedAt":     now,
			},
		}
		opts := options.Update().SetUpsert(true)
		_, updateErrProfile := userProfileColl.UpdateOne(c.Request.Context(), bson.M{"userId": loggedInUserID}, profileUpdate, opts)
		if updateErrProfile != nil {
			c.Error(updateErrProfile)
		}
	}

	// 8. Send the JSON response directly
	c.JSON(http.StatusOK, parsed)
}

type CustomAssessmentRequest struct {
	SkillName string `json:"skillName" binding:"required"`
}

type CustomAssessmentResponse struct {
	IsValid    bool                        `json:"isValid"`
	Assessment *models.GeneratedAssessment `json:"assessment,omitempty"`
}

// HandleCreateCustomAssessment handles generating custom assessment recommendations
func (h *CVHandler) HandleCreateCustomAssessment(c *gin.Context) {
	// 1. Get logged-in user ID
	var loggedInUserID string
	if userVal, exists := c.Get("user"); exists {
		if userMap, ok := userVal.(gin.H); ok {
			if uid, ok := userMap["id"].(string); ok {
				loggedInUserID = uid
			}
		}
	}

	if loggedInUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Validasi saldo token (butuh 1 token untuk custom assessment)
	userColl := database.DB.Collection("user")
	var dbUser models.User
	if err := userColl.FindOne(c.Request.Context(), bson.M{"_id": loggedInUserID}).Decode(&dbUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User tidak ditemukan"})
		return
	}

	if dbUser.TokenBalance == nil || dbUser.TokenBalance.Current < 1 {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": "Saldo kredit Anda tidak mencukupi untuk menambahkan asesmen kustom. Silakan top up kredit terlebih dahulu (Dibutuhkan 1 token).",
		})
		return
	}

	// 2. Parse request body
	var req CustomAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama skill harus diisi"})
		return
	}

	skillName := strings.TrimSpace(req.SkillName)
	if skillName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama skill tidak boleh kosong"})
		return
	}

	// 3. Fetch existing profile
	userProfileColl := database.DB.Collection("userProfile")
	var profile models.UserProfile
	err := userProfileColl.FindOne(c.Request.Context(), bson.M{"userId": loggedInUserID}).Decode(&profile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Profil user tidak ditemukan. Silakan upload CV terlebih dahulu."})
		return
	}

	userLevel := "Mid"
	if profile.CVLevel != nil && *profile.CVLevel != "" {
		userLevel = *profile.CVLevel
	}

	// 4. Prompt Groq to validate and generate assessment recommendation
	systemPrompt := "You are an AI assistant validating and generating custom assessment recommendations.\n" +
		"The candidate wants to generate a custom assessment for the skill/topic: \"" + skillName + "\".\n\n" +
		"First, analyze if the topic is a valid IT, programming language, framework, software tool, design concept, soft skill, or standard professional topic (to avoid nonsensical, offensive, or completely irrelevant topics).\n" +
		"If the topic is NOT valid, nonsensical, or out of mind, return this JSON:\n" +
		"{\n" +
		"  \"isValid\": false\n" +
		"}\n\n" +
		"If the topic is valid, return this JSON:\n" +
		"{\n" +
		"  \"isValid\": true,\n" +
		"  \"assessment\": {\n" +
		"    \"id\": \"A unique slug/ID starting with 'skill-' followed by slugified name\",\n" +
		"    \"type\": \"skill\",\n" +
		"    \"title\": \"Properly capitalized skill/topic title\",\n" +
		"    \"description\": \"Short description of what is tested in this assessment\",\n" +
		"    \"difficulty\": \"Difficulty matching the user level (Junior -> Beginner/Intermediate, Mid -> Intermediate, Senior/Lead -> Advanced). The user level is: " + userLevel + "\",\n" +
		"    \"estimatedTime\": \"45 menit\",\n" +
		"    \"questionCount\": 30,\n" +
		"    \"isRecommended\": true,\n" +
		"    \"category\": \"Suitable category like Frontend, Backend, DevOps, Database, Mobile, Cloud, Design, Security, QA, Data Science, etc.\",\n" +
		"    \"status\": \"available\",\n" +
		"    \"progress\": 0,\n" +
		"    \"sessionId\": \"\",\n" +
		"    \"score\": 0,\n" +
		"    \"level\": \"\"\n" +
		"  }\n" +
		"}\n\n" +
		"Do NOT include any markdown format tags or code block formatting like triple backticks. Return ONLY the raw JSON object."

	groqReq := groq.ChatCompletionRequest{
		Messages: []groq.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: skillName},
		},
		Model: "qwen/qwen3-32b",
		ResponseFormat: &groq.ResponseFormat{
			Type: "json_object",
		},
	}

	resp, err := h.groqClient.CreateChatCompletion(groqReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal terhubung dengan agen validasi"})
		return
	}

	if len(resp.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Agen validasi mengembalikan respon kosong"})
		return
	}

	rawContent := resp.Choices[0].Message.Content
	cleanJSON := strings.TrimSpace(rawContent)

	if strings.HasPrefix(cleanJSON, "```") {
		lines := strings.Split(cleanJSON, "\n")
		var validLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "```") {
				validLines = append(validLines, line)
			}
		}
		cleanJSON = strings.Join(validLines, "\n")
	}
	cleanJSON = strings.TrimSpace(cleanJSON)

	var validationResult CustomAssessmentResponse
	if err := json.Unmarshal([]byte(cleanJSON), &validationResult); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengurai respon validasi"})
		return
	}

	// 5. Fallback message if invalid
	if !validationResult.IsValid || validationResult.Assessment == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Asesmen tidak bisa diproses"})
		return
	}

	// 6. Check for duplicate assessment title/ID
	assessmentToAdd := validationResult.Assessment
	duplicateFound := false
	for _, existing := range profile.CVAssessments {
		if strings.EqualFold(existing.Title, assessmentToAdd.Title) || existing.ID == assessmentToAdd.ID {
			duplicateFound = true
			break
		}
	}

	if duplicateFound {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Asesmen untuk skill ini sudah ada"})
		return
	}

	// Initialize status and metadata fields
	assessmentToAdd.Status = "available"
	assessmentToAdd.Progress = 0
	assessmentToAdd.SessionID = ""
	assessmentToAdd.Score = 0
	assessmentToAdd.Level = ""

	// Kurangi token (current - 1, totalSpent + 1)
	updateToken := bson.M{
		"$inc": bson.M{
			"tokenBalance.current":    -1,
			"tokenBalance.totalSpent": 1,
		},
	}
	_, updateTokenErr := userColl.UpdateOne(c.Request.Context(), bson.M{"_id": loggedInUserID}, updateToken)
	if updateTokenErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memotong saldo token"})
		return
	}

	// 7. Update UserProfile with the new assessment in MongoDB
	profile.CVAssessments = append(profile.CVAssessments, *assessmentToAdd)

	now := time.Now()
	_, updateErr := userProfileColl.UpdateOne(
		c.Request.Context(),
		bson.M{"userId": loggedInUserID},
		bson.M{
			"$set": bson.M{
				"cvAssessments": profile.CVAssessments,
				"updatedAt":     now,
			},
		},
	)
	if updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan rekomendasi asesmen ke profil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Asesmen kustom berhasil ditambahkan",
		"assessment": assessmentToAdd,
	})
}

// saveUploadedFile saves the uploaded file to dst manually without calling os.Chmod on the parent directory
func saveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}
