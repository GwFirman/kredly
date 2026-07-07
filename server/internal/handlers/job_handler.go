package handlers

import (
	"context"
	"kredly/internal/models"
	"kredly/internal/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type JobHandler struct {
	apify *service.ApifyService
	db    *mongo.Database
}

func NewJobHandler(db *mongo.Database) *JobHandler {
	return &JobHandler{
		apify: service.NewApifyService(),
		db:    db,
	}
}

func (h *JobHandler) FetchAndStoreJobs(c *gin.Context) {
	var req models.SearchJobRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userMap, ok := user.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	// Pengecekan token balance
	userColl := h.db.Collection("user")
	var dbUser models.User
	err := userColl.FindOne(c.Request.Context(), bson.M{"_id": userID}).Decode(&dbUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// Validasi saldo token
	if dbUser.TokenBalance == nil || dbUser.TokenBalance.Current < 1 {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": "Saldo kredit Anda tidak mencukupi untuk melakukan pencarian pekerjaan baru. Silakan top up kredit terlebih dahulu.",
		})
		return
	}

	// Kurangi token (current - 1, totalSpent + 1)
	update := bson.M{
		"$inc": bson.M{
			"tokenBalance.current":    -1,
			"tokenBalance.totalSpent": 1,
		},
	}
	_, err = userColl.UpdateOne(c.Request.Context(), bson.M{"_id": userID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memotong saldo token"})
		return
	}

	// Get user's CV role from userProfile to use as search query
	var userProfile models.UserProfile
	err = h.db.Collection("userProfile").FindOne(
		context.Background(),
		bson.M{"userId": userID},
	).Decode(&userProfile)

	// Use CV role as query if available, otherwise use request query or default
	searchQuery := req.Query
	if err == nil && userProfile.CVRole != nil && *userProfile.CVRole != "" {
		searchQuery = *userProfile.CVRole
	} else if searchQuery == "" {
		searchQuery = "Software Developer" // fallback default
	}

	// Build CV data for better job matching
	var cvData *service.CVData
	if err == nil {
		cvData = &service.CVData{
			Role:       userProfile.CVRole,
			Level:      userProfile.CVLevel,
			Skills:     userProfile.CVSkills,
			Experience: userProfile.Experience,
			IsStudent:  userProfile.IsStudent,
			Degree:     userProfile.Degree,
			Summary:    userProfile.CVSummary,
		}
	}

	// Delete existing jobs for this user first to save storage
	_, err = h.db.Collection("job").DeleteMany(
		context.Background(),
		bson.M{"userId": userID},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old jobs"})
		return
	}

	jobs, err := h.apify.SearchAllJobs(searchQuery, req.Location, cvData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	jobDocs := []interface{}{}

	for _, job := range jobs {
		jobDoc := models.Job{
			ID:              uuid.New().String(),
			UserID:          userID,
			Source:          job.Source,
			Title:           job.Title,
			Company:         job.Company,
			CompanyURL:      job.CompanyURL,
			Location:        job.Location,
			RecruiterName:   job.RecruiterName,
			RecruiterURL:    job.RecruiterURL,
			ExperienceLevel: job.ExperienceLevel,
			JobType:         job.JobType,
			Sector:          job.Sector,
			Salary:          job.Salary,
			Description:     job.Description,
			DescriptionHTML: job.DescriptionHTML,
			URL:             job.URL,
			PostedDate:      job.Posted,
			PostedDateExact: job.PostedDate,
			Logo:            job.Logo,
			Promoted:        false,
			EarlyApplicant:  false,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		jobDocs = append(jobDocs, jobDoc)
	}

	if len(jobDocs) > 0 {
		_, err = h.db.Collection("job").InsertMany(context.Background(), jobDocs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Jobs fetched and stored successfully",
		"count":   len(jobDocs),
		"jobs":    jobDocs,
	})
}

func (h *JobHandler) GetUserJobs(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userMap, ok := user.(gin.H)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userID, ok := userMap["id"].(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	cursor, err := h.db.Collection("job").Find(
		context.Background(),
		bson.M{"userId": userID},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(context.Background())

	jobs := []models.Job{}
	if err = cursor.All(context.Background(), &jobs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, jobs)
}
