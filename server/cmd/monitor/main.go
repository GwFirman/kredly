package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"kredly/internal/config"
)

// TokenUsageEvent matches the server event structure
type TokenUsageEvent struct {
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
}

func main() {
	fmt.Println("==================================================")
	fmt.Println("           KREDLY GROQ TOKEN MONITOR              ")
	fmt.Println("==================================================")
	fmt.Println("Press [ENTER] to start recording token usage...")

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	startTime := time.Now()
	fmt.Println("[+] Monitoring started! Waiting for Groq API calls...")
	fmt.Println("[+] Press [ENTER] at any time to stop and print the summary report.\n")

	// Set up HTTP client with no timeout for continuous stream reading
	client := &http.Client{
		Timeout: 0,
	}

	// Load configuration to get the server API URL dynamically
	cfg := config.LoadConfig()
	apiURL := strings.Trim(cfg.APIURL, "\"")

	// Determine the correct base path prefix (e.g. "/api")
	basePath := "/api"
	if os.Getenv("VERCEL") == "1" || os.Getenv("VERCEL_ENV") != "" {
		basePath = ""
	}

	var url string
	apiURLTrimmed := strings.TrimSuffix(apiURL, "/")
	if strings.HasSuffix(apiURLTrimmed, "/api") {
		url = fmt.Sprintf("%s/groq/stream-tokens", apiURLTrimmed)
	} else {
		url = fmt.Sprintf("%s%s/groq/stream-tokens", apiURLTrimmed, basePath)
	}


	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("[-] Error creating request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[-] Error connecting to server stream: %v\n", err)
		fmt.Printf("    Please make sure the Kredly backend server is running on %s!\n", apiURL)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[-] Error: server returned HTTP status %d\n", resp.StatusCode)
		return
	}

	// Variables to store accumulated session stats
	var totalRequests int
	var totalPrompt int
	var totalCompletion int
	var totalTokens int

	// Scan incoming stream events asynchronously in a goroutine
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			// Parse data payload from Server-Sent Event (SSE)
			if strings.HasPrefix(line, "data:") {
				dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				var event TokenUsageEvent
				if err := json.Unmarshal([]byte(dataStr), &event); err == nil {
					totalRequests++
					totalPrompt += event.PromptTokens
					totalCompletion += event.CompletionTokens
					totalTokens += event.TotalTokens

					ts := time.Now().Format("15:04:05")
					fmt.Printf(">> [%s] Model: %-15s | Prompt: %-5d | Compl: %-5d | Total: %d tokens\n",
						ts, event.Model, event.PromptTokens, event.CompletionTokens, event.TotalTokens)
				}
			}
		}
	}()

	// Wait for second enter keypress to terminate
	_, _ = reader.ReadString('\n')

	duration := time.Since(startTime).Round(time.Second)

	fmt.Println("\n[-] Recording stopped.")
	fmt.Println("==================================================")
	fmt.Println("               SESSION SUMMARY REPORT             ")
	fmt.Println("==================================================")
	fmt.Printf("Duration           : %s\n", duration)
	fmt.Printf("Total Requests     : %d\n", totalRequests)
	fmt.Printf("Total Prompt       : %d tokens\n", totalPrompt)
	fmt.Printf("Total Completion   : %d tokens\n", totalCompletion)
	fmt.Printf("Total Tokens Used  : %d tokens\n", totalTokens)
	fmt.Println("==================================================")
	fmt.Println("Goodbye!")
}
