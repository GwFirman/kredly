package models

import (
	"context"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ==========================================
// 1. DEFINISI STRUCT (MODEL DATABASE)
// ==========================================

// TokenBalance menyimpan saldo kredit/token user
type TokenBalance struct {
	Current     int `bson:"current" json:"current"`
	TotalEarned int `bson:"totalEarned" json:"totalEarned"`
	TotalSpent  int `bson:"totalSpent" json:"totalSpent"`
}

type User struct {
	ID            string    `bson:"_id" json:"id"`
	Name          string    `bson:"name" json:"name"`
	Username      *string   `bson:"username,omitempty" json:"username,omitempty"`
	Email         string    `bson:"email" json:"email"`
	EmailVerified bool      `bson:"emailVerified" json:"emailVerified"`
	Image         *string   `bson:"image,omitempty" json:"image,omitempty"`
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt" json:"updatedAt"`

	// Token/Credit balance
	TokenBalance *TokenBalance `bson:"tokenBalance,omitempty" json:"tokenBalance,omitempty"`

	// CV parsed fields
	CVRole     *string    `bson:"cvRole,omitempty" json:"cvRole,omitempty"`
	CVLevel    *string    `bson:"cvLevel,omitempty" json:"cvLevel,omitempty"`
	CVSkills   []string   `bson:"cvSkills,omitempty" json:"cvSkills,omitempty"`
	CVSummary  *string    `bson:"cvSummary,omitempty" json:"cvSummary,omitempty"`
	CVParsedAt *time.Time `bson:"cvParsedAt,omitempty" json:"cvParsedAt,omitempty"`

	// Relasi diabaikan di level database, hanya untuk JSON response jika diperlukan
	Sessions []Session `bson:"-" json:"sessions,omitempty"`
	Accounts []Account `bson:"-" json:"accounts,omitempty"`
}

type ASession struct {
	ID        string    `bson:"_id" json:"id"`
	ExpiresAt time.Time `bson:"expiresAt" json:"expiresAt"`
	Token     string    `bson:"token" json:"token"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
	IPAddress *string   `bson:"ipAddress,omitempty" json:"ipAddress,omitempty"`
	UserAgent *string   `bson:"userAgent,omitempty" json:"userAgent,omitempty"`
	UserID    string    `bson:"userId" json:"userId"`

	User *User `bson:"-" json:"user,omitempty"`
}

type Account struct {
	ID                    string     `bson:"_id" json:"id"`
	AccountID             string     `bson:"accountId" json:"accountId"`
	ProviderID            string     `bson:"providerId" json:"providerId"`
	UserID                string     `bson:"userId" json:"userId"`
	AccessToken           *string    `bson:"accessToken,omitempty" json:"accessToken,omitempty"`
	RefreshToken          *string    `bson:"refreshToken,omitempty" json:"refreshToken,omitempty"`
	IDToken               *string    `bson:"idToken,omitempty" json:"idToken,omitempty"`
	AccessTokenExpiresAt  *time.Time `bson:"accessTokenExpiresAt,omitempty" json:"accessTokenExpiresAt,omitempty"`
	RefreshTokenExpiresAt *time.Time `bson:"refreshTokenExpiresAt,omitempty" json:"refreshTokenExpiresAt,omitempty"`
	Scope                 *string    `bson:"scope,omitempty" json:"scope,omitempty"`
	Password              *string    `bson:"password,omitempty" json:"password,omitempty"`
	CreatedAt             time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt             time.Time  `bson:"updatedAt" json:"updatedAt"`

	User *User `bson:"-" json:"user,omitempty"`
}

type Verification struct {
	ID         string     `bson:"_id" json:"id"`
	Identifier string     `bson:"identifier" json:"identifier"`
	Value      string     `bson:"value" json:"value"`
	ExpiresAt  time.Time  `bson:"expiresAt" json:"expiresAt"`
	CreatedAt  *time.Time `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt  *time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

// UserProfile menyimpan data onboarding user (CV, pengalaman, status student)
type UserProfile struct {
	ID                      string                `bson:"_id" json:"id"`
	UserID                  string                `bson:"userId" json:"userId"`                     // Relasi ke User._id
	CVFileName              string                `bson:"cvFileName" json:"cvFileName"`             // Nama file CV
	CVFilePath              string                `bson:"cvFilePath" json:"cvFilePath"`             // Path/URL file CV di server
	Experience              string                `bson:"experience" json:"experience"`             // "below-1", "1-2", "3-5", "not-working"
	IsStudent               bool                  `bson:"isStudent" json:"isStudent"`               // true/false
	Degree                  *string               `bson:"degree,omitempty" json:"degree,omitempty"` // Jurusan (opsional, jika student)
	CVRole                  *string               `bson:"cvRole,omitempty" json:"cvRole,omitempty"`
	CVLevel                 *string               `bson:"cvLevel,omitempty" json:"cvLevel,omitempty"`
	CVSkills                []string              `bson:"cvSkills,omitempty" json:"cvSkills,omitempty"`
	CVSummary               *string               `bson:"cvSummary,omitempty" json:"cvSummary,omitempty"`
	CVAssessments           []GeneratedAssessment `bson:"cvAssessments,omitempty" json:"cvAssessments,omitempty"`
	RoleAssessmentCompleted bool                  `bson:"roleAssessmentCompleted" json:"roleAssessmentCompleted"`
	CreatedAt               time.Time             `bson:"createdAt" json:"createdAt"`
	UpdatedAt               time.Time             `bson:"updatedAt" json:"updatedAt"`

	User *User `bson:"-" json:"user,omitempty"` // Relasi untuk JSON response
}

type GeneratedAssessment struct {
	ID            string     `bson:"id" json:"id"`
	Type          string     `bson:"type" json:"type"` // "general" or "skill"
	Title         string     `bson:"title" json:"title"`
	Description   string     `bson:"description,omitempty" json:"description,omitempty"`
	Difficulty    string     `bson:"difficulty" json:"difficulty"`
	EstimatedTime string     `bson:"estimatedTime" json:"estimatedTime"`
	QuestionCount int        `bson:"questionCount" json:"questionCount"`
	Topics        []string   `bson:"topics,omitempty" json:"topics,omitempty"`
	IsRecommended bool       `bson:"isRecommended" json:"isRecommended"`
	Category      string     `bson:"category" json:"category"`
	Status        string     `bson:"status" json:"status"` // "available", "in-progress", "completed"
	Progress      int        `bson:"progress,omitempty" json:"progress,omitempty"`
	SessionID     string     `bson:"sessionId,omitempty" json:"sessionId,omitempty"`
	Score         int        `bson:"score,omitempty" json:"score,omitempty"`
	Level         string     `bson:"level,omitempty" json:"level,omitempty"`
	ExpiresAt     *time.Time `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
}

type Job struct {
	ID              string    `bson:"_id" json:"id"`
	UserID          string    `bson:"userId" json:"userId"`
	Source          string    `bson:"source" json:"source"` // "linkedin", "indeed", "glassdoor", "upwork"
	Title           string    `bson:"title" json:"title"`
	Company         string    `bson:"company" json:"company"`
	CompanyURL      *string   `bson:"companyUrl,omitempty" json:"companyUrl,omitempty"`
	Location        string    `bson:"location" json:"location"`
	RecruiterName   *string   `bson:"recruiterName,omitempty" json:"recruiterName,omitempty"`
	RecruiterURL    *string   `bson:"recruiterUrl,omitempty" json:"recruiterUrl,omitempty"`
	ExperienceLevel *string   `bson:"experienceLevel,omitempty" json:"experienceLevel,omitempty"`
	JobType         *string   `bson:"jobType,omitempty" json:"type,omitempty"`
	Sector          *string   `bson:"sector,omitempty" json:"sector,omitempty"`
	Salary          *string   `bson:"salary,omitempty" json:"salary,omitempty"`
	Description     *string   `bson:"description,omitempty" json:"description,omitempty"`
	DescriptionHTML *string   `bson:"descriptionHtml,omitempty" json:"descriptionHtml,omitempty"`
	URL             *string   `bson:"url,omitempty" json:"url,omitempty"`
	PostedDate      *string   `bson:"postedDate,omitempty" json:"postedTime,omitempty"`
	PostedDateExact *string   `bson:"postedDateExact,omitempty" json:"postedDate,omitempty"`
	Logo            *string   `bson:"logo,omitempty" json:"logo,omitempty"`
	Promoted        bool      `bson:"promoted" json:"promoted"`
	EarlyApplicant  bool      `bson:"earlyApplicant" json:"earlyApplicant"`
	CreatedAt       time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time `bson:"updatedAt" json:"updatedAt"`

	User *User `bson:"-" json:"user,omitempty"`
}

type SocialLinks struct {
	Linkedin  string `bson:"linkedin" json:"linkedin"`
	Github    string `bson:"github" json:"github"`
	Portfolio string `bson:"portfolio" json:"portfolio"`
	Twitter   string `bson:"twitter" json:"twitter"`
}

type PublicProfileSettings struct {
	ID               string      `bson:"_id" json:"id"`
	UserID           string      `bson:"userId" json:"userId"`
	IsPublic         bool        `bson:"isPublic" json:"isPublic"`
	Headline         string      `bson:"headline" json:"headline"`
	Bio              string      `bson:"bio" json:"bio"`
	ShowCertificates bool        `bson:"showCertificates" json:"showCertificates"`
	ShowAssessments  bool        `bson:"showAssessments" json:"showAssessments"`
	ShowSkills       bool        `bson:"showSkills" json:"showSkills"`
	ShowCVData       bool        `bson:"showCVData" json:"showCVData"`
	SocialLinks      SocialLinks `bson:"socialLinks" json:"socialLinks"`
	CreatedAt        time.Time   `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time   `bson:"updatedAt" json:"updatedAt"`
}

// ==========================================
// 2. SETUP UNIQUE INDEXES
// ==========================================

// SetupIndexes berfungsi mengeksekusi pembuatan index unik (pengganti @@unique di Prisma)
func SetupIndexes(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Index unik untuk User (Email)
	userCollection := db.Collection("user")
	_, err := userCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		// Cek apakah errornya karena index sudah ada (conflict)
		if strings.Contains(err.Error(), "IndexOptionsConflict") || strings.Contains(err.Error(), "already exists") {
			log.Println("⚠️ Index unik untuk User Email sudah ada, proses diabaikan.")
		} else {
			log.Printf("❌ Gagal membuat index unik untuk User Email: %v", err)
			return err
		}
	}

	// 1b. Index unik untuk User (Username)
	_, err = userCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true), // sparse: true karena username bisa null
	})
	if err != nil {
		if strings.Contains(err.Error(), "IndexOptionsConflict") || strings.Contains(err.Error(), "already exists") {
			log.Println("⚠️ Index unik untuk User Username sudah ada, proses diabaikan.")
		} else {
			log.Printf("❌ Gagal membuat index unik untuk User Username: %v", err)
			return err
		}
	}

	// 2. Index unik untuk Session (Token)
	sessionCollection := db.Collection("session")
	_, err = sessionCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		// Cek apakah errornya karena index sudah ada (conflict)
		if strings.Contains(err.Error(), "IndexOptionsConflict") || strings.Contains(err.Error(), "already exists") {
			log.Println("⚠️ Index unik untuk Session Token sudah ada, proses diabaikan.")
		} else {
			log.Printf("❌ Gagal membuat index unik untuk Session Token: %v", err)
			return err
		}
	}

	// 3. Index unik untuk UserProfile (UserID) - satu user hanya punya satu profile
	userProfileCollection := db.Collection("userProfile")
	_, err = userProfileCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		if strings.Contains(err.Error(), "IndexOptionsConflict") || strings.Contains(err.Error(), "already exists") {
			log.Println("⚠️ Index unik untuk UserProfile UserID sudah ada, proses diabaikan.")
		} else {
			log.Printf("❌ Gagal membuat index unik untuk UserProfile UserID: %v", err)
			return err
		}
	}

	// 4. Index unik untuk PublicProfileSettings (UserID)
	publicProfileSettingsCollection := db.Collection("publicProfileSettings")
	_, err = publicProfileSettingsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		if strings.Contains(err.Error(), "IndexOptionsConflict") || strings.Contains(err.Error(), "already exists") {
			log.Println("⚠️ Index unik untuk PublicProfileSettings UserID sudah ada, proses diabaikan.")
		} else {
			log.Printf("❌ Gagal membuat index unik untuk PublicProfileSettings UserID: %v", err)
			return err
		}
	}

	log.Println("✅ Berhasil mensinkronisasi semua index MongoDB!")
	return nil
}
