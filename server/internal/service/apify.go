package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type ApifyService struct{}

func NewApifyService() *ApifyService {
	return &ApifyService{}
}

type CVData struct {
	Role       *string
	Level      *string
	Skills     []string
	Experience string
	IsStudent  bool
	Degree     *string
	Summary    *string
}

type JobResult struct {
	ID              *string
	Source          string
	Title           string
	Company         string
	CompanyURL      *string
	Location        string
	RecruiterName   *string
	RecruiterURL    *string
	ExperienceLevel *string
	JobType         *string
	Sector          *string
	Salary          *string
	URL             *string
	Posted          *string
	PostedDate      *string
	Logo            *string
	Description     *string
	DescriptionHTML *string
}

func (s *ApifyService) SearchAllJobs(
	query string,
	location string,
	cvData *CVData,
) ([]JobResult, error) {
	token := os.Getenv("APIFY_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("APIFY_TOKEN not set")
	}

	// Generate random limits for each platform (3-5 jobs each) totaling 15
	limits := generateRandomLimits(15, 4)

	var wg sync.WaitGroup
	results := make([][]JobResult, 4)
	errors := make([]error, 4)

	scrapers := []struct {
		name  string
		fn    func(string, string, string, int, *CVData) ([]JobResult, error)
		index int
		limit int
	}{
		{"linkedin", s.searchLinkedIn, 0, limits[0]},
		{"indeed", s.searchIndeed, 1, limits[1]},
		{"glassdoor", s.searchGlassdoor, 2, limits[2]},
		{"upwork", s.searchUpwork, 3, limits[3]},
	}

	fmt.Printf("\n=== Starting job search ===\n")
	fmt.Printf("Query: %s, Location: %s\n", query, location)
	fmt.Printf("Limits: LinkedIn=%d, Indeed=%d, Glassdoor=%d, Upwork=%d\n",
		limits[0], limits[1], limits[2], limits[3])

	for _, scraper := range scrapers {
		wg.Add(1)
		go func(name string, fn func(string, string, string, int, *CVData) ([]JobResult, error), idx int, limit int) {
			defer wg.Done()
			fmt.Printf("[%s] Starting scrape (limit: %d)...\n", name, limit)
			jobs, err := fn(token, query, location, limit, cvData)
			if err != nil {
				fmt.Printf("[%s] ERROR: %v\n", name, err)
				errors[idx] = err
				return
			}
			fmt.Printf("[%s] SUCCESS: %d jobs fetched\n", name, len(jobs))
			results[idx] = jobs
		}(scraper.name, scraper.fn, scraper.index, scraper.limit)
	}

	wg.Wait()

	allJobs := []JobResult{}
	fmt.Printf("\n=== Results Summary ===\n")
	for i, jobs := range results {
		scraperName := scrapers[i].name
		if errors[i] != nil {
			fmt.Printf("[%s] FAILED: %v\n", scraperName, errors[i])
			continue
		}
		fmt.Printf("[%s] %d jobs\n", scraperName, len(jobs))
		allJobs = append(allJobs, jobs...)
	}
	fmt.Printf("Total jobs: %d\n\n", len(allJobs))

	if len(allJobs) == 0 {
		errorMessages := ""
		for i, err := range errors {
			if err != nil {
				errorMessages += fmt.Sprintf("%s: %v; ", scrapers[i].name, err)
			}
		}
		return nil, fmt.Errorf("all scrapers failed: %s", errorMessages)
	}

	return allJobs, nil
}

// generateRandomLimits generates random limits that sum to total
// Each limit is between 3 and 5
func generateRandomLimits(total int, count int) []int {
	limits := make([]int, count)
	remaining := total

	for i := 0; i < count-1; i++ {
		// Each platform gets between 3-5 jobs, but ensure we can still reach the total
		minForThis := 3
		maxForThis := 5
		minNeededForRest := (count - i - 1) * 3
		maxNeededForRest := (count - i - 1) * 5

		// Adjust bounds to ensure total can be reached
		if remaining-maxForThis < minNeededForRest {
			minForThis = remaining - maxNeededForRest
		}
		if remaining-minForThis > maxNeededForRest {
			maxForThis = remaining - minNeededForRest
		}

		// Clamp to 3-5 range
		if minForThis < 3 {
			minForThis = 3
		}
		if maxForThis > 5 {
			maxForThis = 5
		}
		if maxForThis < minForThis {
			maxForThis = minForThis
		}

		// Random value in the adjusted range
		limits[i] = minForThis + (remaining % (maxForThis - minForThis + 1))
		remaining -= limits[i]
	}

	// Last platform gets whatever is left
	limits[count-1] = remaining

	return limits
}

// LinkedIn Jobs Scraper - hKByXkMQaC5Qt9UMN
func (s *ApifyService) searchLinkedIn(
	token string,
	query string,
	location string,
	limit int,
	cvData *CVData,
) ([]JobResult, error) {
	url := fmt.Sprintf(
		"https://api.apify.com/v2/acts/hKByXkMQaC5Qt9UMN/run-sync-get-dataset-items?token=%s",
		token,
	)

	linkedinSearchURL := fmt.Sprintf(
		"https://www.linkedin.com/jobs/search/?keywords=%s&location=%s&position=1&pageNum=0",
		query,
		location,
	)

	// LinkedIn requires minimum count of 10
	linkedinCount := limit
	if linkedinCount < 10 {
		linkedinCount = 10
	}

	payload := map[string]interface{}{
		"urls":            []string{linkedinSearchURL},
		"scrapeCompany":   true,
		"count":           linkedinCount,
		"splitByLocation": false,
	}

	items, err := s.callApify(url, payload)
	if err != nil {
		return nil, err
	}

	jobs := []JobResult{}
	for _, item := range items {
		job := JobResult{
			Source:   "linkedin",
			Title:    getStringField(item, "title"),
			Company:  getStringField(item, "companyName"),
			Location: getStringField(item, "location"),
		}

		// Skip jobs without required fields
		if job.Title == "" || job.Company == "" {
			fmt.Printf("[linkedin] ⚠️  Skipping incomplete job - missing title or company\n")
			continue
		}

		if id := getStringField(item, "id"); id != "" {
			job.ID = &id
		}
		if companyURL := getStringField(item, "companyLinkedinUrl"); companyURL != "" {
			job.CompanyURL = &companyURL
		}
		if logo := getStringField(item, "companyLogo"); logo != "" {
			job.Logo = &logo
		}
		if recruiterName := getStringField(item, "jobPosterName"); recruiterName != "" {
			job.RecruiterName = &recruiterName
		}
		if recruiterURL := getStringField(item, "jobPosterProfileUrl"); recruiterURL != "" {
			job.RecruiterURL = &recruiterURL
		}
		if experienceLevel := getStringField(item, "seniorityLevel"); experienceLevel != "" {
			job.ExperienceLevel = &experienceLevel
		}
		if jobType := getStringField(item, "employmentType"); jobType != "" {
			job.JobType = &jobType
		}
		if industries := getStringField(item, "industries"); industries != "" {
			job.Sector = &industries
		}
		if salaryInfo, ok := item["salaryInfo"].([]interface{}); ok && len(salaryInfo) > 0 {
			salaryParts := []string{}
			for _, s := range salaryInfo {
				if str, ok := s.(string); ok {
					salaryParts = append(salaryParts, str)
				}
			}
			if len(salaryParts) > 0 {
				salaryText := salaryParts[0]
				if len(salaryParts) > 1 {
					salaryText = fmt.Sprintf("%s - %s", salaryParts[0], salaryParts[1])
				}
				job.Salary = &salaryText
			}
		}
		if jobURL := getStringField(item, "link"); jobURL != "" {
			job.URL = &jobURL
		}
		if postedDate := getStringField(item, "postedAt"); postedDate != "" {
			job.PostedDate = &postedDate
		}
		if description := getStringField(item, "descriptionText"); description != "" {
			job.Description = &description
		}
		if descriptionHTML := getStringField(item, "descriptionHtml"); descriptionHTML != "" {
			job.DescriptionHTML = &descriptionHTML
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Indeed Jobs Scraper - hMvNSpz3JnHgl5jkh
func (s *ApifyService) searchIndeed(
	token string,
	query string,
	location string,
	limit int,
	cvData *CVData,
) ([]JobResult, error) {
	url := fmt.Sprintf(
		"https://api.apify.com/v2/acts/hMvNSpz3JnHgl5jkh/run-sync-get-dataset-items?token=%s",
		token,
	)

	// Extract country from location or default to US
	country := "US"
	if location != "" {
		// Simple heuristic for country extraction
		// Default to US for now
		country = "US"
	}

	payload := map[string]interface{}{
		"position":            query,
		"maxItems":            limit,
		"country":             country,
		"location":            location,
		"saveOnlyUniqueItems": true,
	}

	items, err := s.callApify(url, payload)
	if err != nil {
		return nil, err
	}

	jobs := []JobResult{}
	for _, item := range items {
		job := JobResult{
			Source:   "indeed",
			Title:    getStringField(item, "positionName"),
			Company:  getStringField(item, "company"),
			Location: getStringField(item, "location"),
		}

		fmt.Printf("[Indeed] Parsing job - Title: %s, Company: %s, Location: %s\n",
			job.Title, job.Company, job.Location)

		// Skip jobs without required fields
		if job.Title == "" || job.Company == "" {
			fmt.Printf("[Indeed] ⚠️  Skipping incomplete job - missing title or company\n")
			continue
		}

		// Extract ID
		if id := getStringField(item, "id"); id != "" {
			job.ID = &id
			fmt.Printf("[Indeed] - ID: %s\n", id)
		}

		// Extract logo
		if logo := getStringField(item, "companyLogo"); logo != "" {
			job.Logo = &logo
			fmt.Printf("[Indeed] - Logo: %s\n", logo)
		}

		// Extract salary
		if salary := getStringField(item, "salary"); salary != "" {
			job.Salary = &salary
			fmt.Printf("[Indeed] - Salary: %s\n", salary)
		}

		// Extract job type from jobType array
		if jobTypes, ok := item["jobType"].([]interface{}); ok && len(jobTypes) > 0 {
			if jobTypeStr, ok := jobTypes[0].(string); ok {
				job.JobType = &jobTypeStr
				fmt.Printf("[Indeed] - JobType: %s\n", jobTypeStr)
			}
		}

		// Extract URL
		if jobUrl := getStringField(item, "url"); jobUrl != "" {
			job.URL = &jobUrl
			fmt.Printf("[Indeed] - URL: %s\n", jobUrl)
		}

		// Extract description
		if description := getStringField(item, "description"); description != "" {
			job.Description = &description
			fmt.Printf("[Indeed] - Description length: %d chars\n", len(description))
		}
		if descriptionHTML := getStringField(item, "descriptionHTML"); descriptionHTML != "" {
			job.DescriptionHTML = &descriptionHTML
			fmt.Printf("[Indeed] - DescriptionHTML length: %d chars\n", len(descriptionHTML))
		}

		// Extract posted date
		if postedAt := getStringField(item, "postedAt"); postedAt != "" {
			job.Posted = &postedAt
			fmt.Printf("[Indeed] - PostedAt: %s\n", postedAt)
		}

		// Extract scraped date as fallback for posted date
		if scrapedAt := getStringField(item, "scrapedAt"); scrapedAt != "" && job.PostedDate == nil {
			job.PostedDate = &scrapedAt
			fmt.Printf("[Indeed] - ScrapedAt: %s\n", scrapedAt)
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Glassdoor Jobs Scraper - bYSAbQqxwImLaf2nb
func (s *ApifyService) searchGlassdoor(
	token string,
	query string,
	location string,
	limit int,
	cvData *CVData,
) ([]JobResult, error) {
	url := fmt.Sprintf(
		"https://api.apify.com/v2/acts/bYSAbQqxwImLaf2nb/run-sync-get-dataset-items?token=%s",
		token,
	)

	// Build keywords array from query
	keywords := []string{query}

	// Build resume keywords from CV skills for better matching
	resumeKeywords := []map[string]interface{}{}
	if cvData != nil && len(cvData.Skills) > 0 {
		for _, skill := range cvData.Skills {
			resumeKeywords = append(resumeKeywords, map[string]interface{}{
				"keyword": skill,
			})
		}
	}

	// Extract country from location or default to United States
	country := "United States"
	if location != "" {
		// Simple heuristic: if location contains common country patterns
		// For now, default to United States
		country = "United States"
	}

	payload := map[string]interface{}{
		"keywords":             keywords,
		"country":              country,
		"location":             location,
		"datePosted":           "14", // Last 14 days
		"saveOnlyUniqueItems":  true,
		"deepSearch":           true,
		"includeNoSalaryJob":   true,
		"maxItems":             limit,
		"resumeKeywords":       resumeKeywords,
	}

	items, err := s.callApify(url, payload)
	if err != nil {
		return nil, err
	}

	jobs := []JobResult{}
	for _, item := range items {
		// Extract company info from nested company object
		companyName := ""
		var companyURL *string
		var logo *string
		if company, ok := item["company"].(map[string]interface{}); ok {
			companyName = getStringFieldFromMap(company, "companyName")
			if pageUrl := getStringFieldFromMap(company, "companyPageUrl"); pageUrl != "" {
				companyURL = &pageUrl
			}
			if logoUrl := getStringFieldFromMap(company, "logoImgUrl"); logoUrl != "" {
				logo = &logoUrl
			}
		}

		// Extract location from separate fields
		locationCity := getStringField(item, "location_city")
		locationState := getStringField(item, "location_state")
		locationStr := locationCity
		if locationState != "" {
			locationStr = fmt.Sprintf("%s, %s", locationCity, locationState)
		}

		job := JobResult{
			Source:    "glassdoor",
			Title:     getStringField(item, "title"),
			Company:   companyName,
			Location:  locationStr,
			CompanyURL: companyURL,
			Logo:      logo,
		}

		// Skip jobs without required fields
		if job.Title == "" || job.Company == "" {
			fmt.Printf("[glassdoor] ⚠️  Skipping incomplete job - missing title or company\n")
			continue
		}

		// Extract ID from key field
		if key := getStringField(item, "key"); key != "" {
			job.ID = &key
		}

		// Extract experience level from experienceRequired array
		if expRequired, ok := item["experienceRequired"].([]interface{}); ok && len(expRequired) > 0 {
			if expStr, ok := expRequired[0].(string); ok {
				experienceLevel := fmt.Sprintf("%s+ years", expStr)
				job.ExperienceLevel = &experienceLevel
			}
		}

		// Extract job type from jobTypes array
		if jobTypes, ok := item["jobTypes"].([]interface{}); ok && len(jobTypes) > 0 {
			if jobTypeStr, ok := jobTypes[0].(string); ok {
				job.JobType = &jobTypeStr
			}
		}

		// Extract sector from jobCategory or attributes
		if category := getStringField(item, "jobCategory"); category != "" {
			job.Sector = &category
		} else if attributes, ok := item["attributes"].([]interface{}); ok && len(attributes) > 0 {
			// Join first 3 attributes as sector
			skills := []string{}
			for i := 0; i < len(attributes) && i < 3; i++ {
				if attr, ok := attributes[i].(string); ok {
					skills = append(skills, attr)
				}
			}
			if len(skills) > 0 {
				sector := skills[0]
				for i := 1; i < len(skills); i++ {
					sector += fmt.Sprintf(", %s", skills[i])
				}
				job.Sector = &sector
			}
		}

		// Extract salary from baseSalary fields
		if minSalary, okMin := item["baseSalary_min"].(float64); okMin {
			if maxSalary, okMax := item["baseSalary_max"].(float64); okMax {
				currency := getStringField(item, "salary_currency")
				if currency == "" {
					currency = "USD"
				}
				period := getStringField(item, "salary_period")
				salaryText := fmt.Sprintf("%s %.0f - %.0f (%s)", currency, minSalary, maxSalary, period)
				job.Salary = &salaryText
			}
		}

		// Extract URL from jobUrl
		if jobUrl := getStringField(item, "jobUrl"); jobUrl != "" {
			job.URL = &jobUrl
		}

		// Extract description from description_text
		if description := getStringField(item, "description_text"); description != "" {
			job.Description = &description
		}
		if descriptionHTML := getStringField(item, "description_html"); descriptionHTML != "" {
			job.DescriptionHTML = &descriptionHTML
		}

		// Extract posted date from datePublished
		if datePublished := getStringField(item, "datePublished"); datePublished != "" {
			job.PostedDate = &datePublished
		}

		// Calculate relative posted time from ageInDays
		if ageInDays, ok := item["ageInDays"].(float64); ok {
			days := int(ageInDays)
			posted := fmt.Sprintf("%d days ago", days)
			job.Posted = &posted
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Upwork Jobs Scraper - YdYsB7rsRY0EUb1lP
func (s *ApifyService) searchUpwork(
	token string,
	query string,
	location string,
	limit int,
	cvData *CVData,
) ([]JobResult, error) {
	url := fmt.Sprintf(
		"https://api.apify.com/v2/acts/YdYsB7rsRY0EUb1lP/run-sync-get-dataset-items?token=%s",
		token,
	)

	// Build date range for recent jobs (last 14 days)
	now := time.Now()
	fromDate := now.AddDate(0, 0, -14).Format("2006-01-02")
	toDate := now.Format("2006-01-02")

	// Build job categories from CV role
	jobCategories := []string{"Web Development"}
	if cvData != nil && cvData.Role != nil && *cvData.Role != "" {
		jobCategories = mapRoleToCategories(*cvData.Role)
	}

	// Build keywords from CV skills and query
	keywords := []string{}
	if cvData != nil && len(cvData.Skills) > 0 {
		keywords = append(keywords, cvData.Skills...)
	}

	// Map CV level to budget rates
	minHourlyRate := "20"
	maxHourlyRate := "100"
	if cvData != nil && cvData.Level != nil {
		_, minHourlyRate, maxHourlyRate = mapLevelToUpworkParams(*cvData.Level)
	}

	payload := map[string]interface{}{
		"limit":    limit,
		"fromDate": fromDate,
		"toDate":   toDate,
		"jobCategories": jobCategories,
		"includeKeywords.keywords":       keywords,
		"includeKeywords.matchTitle":      true,
		"includeKeywords.matchDescription": true,
		"includeKeywords.matchSkills":     true,
		"excludeKeywords.keywords":        []string{},
		"excludeKeywords.matchTitle":      true,
		"excludeKeywords.matchDescription": true,
		"excludeKeywords.matchSkills":     true,
		"budget.allowUnspecifiedBudget":   false,
		"budget.hourlyRate.min":           minHourlyRate,
		"budget.hourlyRate.max":           maxHourlyRate,
		"budget.avgHourlyRate.min":        minHourlyRate,
		"budget.avgHourlyRate.max":        maxHourlyRate,
		"budget.fixedPrice.min":           "100",
		"budget.fixedPrice.max":           "50000",
		"budget.connectsPrice.min":        1,
		"budget.connectsPrice.max":        16,
		"budget.jobDurations": []string{
			"UNSPECIFIED", "UP_TO_ONE_MONTH", "UP_TO_THREE_MONTHS",
			"UP_TO_SIX_MONTHS", "MORE_THAN_SIX_MONTHS",
		},
		"budget.hourlyWorkloads": []string{
			"UNSPECIFIED", "LESS_THAN_30_HOURS", "MORE_THAN_30_HOURS",
		},
		"budget.noAvgHourlyRatePaid":   false,
		"budget.noHireRate":             false,
		"budget.onlyContractToHire":     false,
		"budget.minClientHireRate":      0,
		"client.companySizeRange": []string{
			"UNSPECIFIED", "SOLO_ENTERPRENEUR", "UP_TO_10_EMPLOYEES",
			"UP_TO_100_EMPOLOYEES", "UP_TO_500_EMPLOYEES",
			"UP_TO_1K_EMPLOYEES", "MORE_THAN_1K_EMPLOYEES",
		},
		"client.descriptionLanguage.exclude": []string{},
		"client.descriptionLanguage.include": []string{},
		"client.hireHistory":                 []string{"NONE", "UP_TO", "MORE_THAN"},
		"jobIds":                             []string{},
		"addons.enableClientDetails":         false,
		"addons.enableClientActivity":        false,
		"addons.enableJobAttachments":        false,
		"notifications.shouldSendRunMetadata": true,
		"notifications.limit":                 3,
	}

	items, err := s.callApify(url, payload)
	if err != nil {
		return nil, err
	}

	jobs := []JobResult{}
	for _, item := range items {
		// Extract client country
		clientCountry := "Upwork Client"
		if client, ok := item["client"].(map[string]interface{}); ok {
			if countryCode := getStringFieldFromMap(client, "countryCode"); countryCode != "" {
				clientCountry = fmt.Sprintf("Client from %s", countryCode)
			}
		}

		job := JobResult{
			Source:   "upwork",
			Title:    getStringField(item, "title"),
			Company:  clientCountry,
			Location: "Remote",
		}

		// Skip jobs without required fields
		if job.Title == "" || job.Company == "" {
			fmt.Printf("[upwork] ⚠️  Skipping incomplete job - missing title or company\n")
			continue
		}

		// Extract ID from uid
		if uid := getStringField(item, "uid"); uid != "" {
			job.ID = &uid
		}

		// Extract experience level from vendor.experienceLevel
		if vendor, ok := item["vendor"].(map[string]interface{}); ok {
			if expLevel := getStringFieldFromMap(vendor, "experienceLevel"); expLevel != "" {
				readableLevel := mapUpworkExperienceToReadable(expLevel)
				job.ExperienceLevel = &readableLevel
			}
		}

		// Extract category as sector
		if category := getStringField(item, "category"); category != "" {
			job.Sector = &category
		}

		// Extract budget
		if budget, ok := item["budget"].(map[string]interface{}); ok {
			var salaryText string
			if hourlyRate, ok := budget["hourlyRate"].(map[string]interface{}); ok {
				if min, okMin := hourlyRate["min"].(float64); okMin {
					if max, okMax := hourlyRate["max"].(float64); okMax {
						salaryText = fmt.Sprintf("$%.0f - $%.0f/hr", min, max)
						job.Salary = &salaryText
					}
				}
			} else if fixedBudget, ok := budget["fixedBudget"].(float64); ok {
				salaryText = fmt.Sprintf("$%.0f (Fixed)", fixedBudget)
				job.Salary = &salaryText
			}
		}

		// Extract URL from externalLink
		if externalLink := getStringField(item, "externalLink"); externalLink != "" {
			job.URL = &externalLink
		}

		// Extract description
		if description := getStringField(item, "description"); description != "" {
			job.Description = &description
		}

		// Extract posted date from createdAt
		if createdAt := getStringField(item, "createdAt"); createdAt != "" {
			job.PostedDate = &createdAt
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// mapRoleToCategories maps CV role to Upwork job categories
func mapRoleToCategories(role string) []string {
	roleMap := map[string][]string{
		"web developer":      {"Web Development"},
		"frontend developer": {"Web Development"},
		"backend developer":  {"Web Development"},
		"full stack":         {"Web Development"},
		"mobile developer":   {"Mobile Development"},
		"ios developer":      {"Mobile Development"},
		"android developer":  {"Mobile Development"},
		"designer":           {"All - Design & Creative"},
		"ui designer":        {"All - Design & Creative"},
		"ux designer":        {"All - Design & Creative"},
		"data scientist":     {"Data Science & Analytics"},
		"data analyst":       {"Data Science & Analytics"},
	}

	// Normalize role to lowercase for matching
	normalizedRole := ""
	for key := range roleMap {
		if len(role) > 0 && len(key) > 0 {
			if role == key || fmt.Sprintf("%s", role) == key {
				normalizedRole = key
				break
			}
		}
	}

	if categories, ok := roleMap[normalizedRole]; ok {
		return categories
	}

	// Default to Web Development
	return []string{"Web Development"}
}

// mapLevelToUpworkParams maps CV level to Upwork experience level and budget range
func mapLevelToUpworkParams(level string) (string, string, string) {
	levelMap := map[string]struct {
		experience string
		minRate    string
		maxRate    string
	}{
		"entry":        {"ENTRY_LEVEL", "10", "40"},
		"junior":       {"ENTRY_LEVEL", "10", "40"},
		"mid":          {"INTERMEDIATE", "30", "80"},
		"intermediate": {"INTERMEDIATE", "30", "80"},
		"senior":       {"EXPERT", "60", "150"},
		"expert":       {"EXPERT", "60", "150"},
		"lead":         {"EXPERT", "80", "200"},
	}

	// Normalize level to lowercase for matching
	normalizedLevel := ""
	for key := range levelMap {
		if len(level) > 0 && len(key) > 0 {
			if level == key || fmt.Sprintf("%s", level) == key {
				normalizedLevel = key
				break
			}
		}
	}

	if params, ok := levelMap[normalizedLevel]; ok {
		return params.experience, params.minRate, params.maxRate
	}

	// Default to intermediate
	return "INTERMEDIATE", "30", "80"
}

// mapUpworkExperienceToReadable maps Upwork experience levels to readable format
func mapUpworkExperienceToReadable(upworkLevel string) string {
	levelMap := map[string]string{
		"ENTRY_LEVEL":  "Entry Level",
		"INTERMEDIATE": "Intermediate",
		"EXPERT":       "Expert",
	}

	if readable, ok := levelMap[upworkLevel]; ok {
		return readable
	}
	return upworkLevel
}

func (s *ApifyService) callApify(
	url string,
	payload map[string]interface{},
) ([]map[string]interface{}, error) {
	bodyBytes, _ := json.Marshal(payload)

	fmt.Printf("\n--- Apify API Call ---\n")
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Payload: %s\n", string(bodyBytes))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		fmt.Printf("ERROR creating request: %v\n", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("HTTP Status: %d\n", resp.StatusCode)

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("ERROR Response Body: %s\n", string(body))
		return nil, fmt.Errorf("apify returned status %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response size: %d bytes\n", len(body))

	var items []map[string]interface{}
	err = json.Unmarshal(body, &items)
	if err != nil {
		fmt.Printf("ERROR parsing JSON: %v\n", err)
		fmt.Printf("Response body (first 500 chars): %s\n", string(body[:min(500, len(body))]))
		return nil, err
	}

	fmt.Printf("Items returned: %d\n", len(items))
	fmt.Printf("--- End API Call ---\n\n")

	return items, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getStringField(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getStringFieldVariants(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val := getStringField(data, key); val != "" {
			return val
		}
	}
	return ""
}

// getNestedStringField extracts a string from a nested object (e.g., employer.name)
func getNestedStringField(data map[string]interface{}, parentKey string, childKey string) string {
	if parent, ok := data[parentKey].(map[string]interface{}); ok {
		return getStringField(parent, childKey)
	}
	return ""
}

// getStringFieldFromMap extracts a string from a map (helper for already-extracted nested objects)
func getStringFieldFromMap(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
