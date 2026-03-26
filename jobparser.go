package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

func ExtractJobDetails(subject, body string) (string, string, string, bool, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	prompt := `You are reviewing an email to determine if it is related to a job application.

First decide: is this genuinely a job application email? This includes:
- Application confirmations / receipts
- Interview invitations or scheduling
- Job offers
- Rejections or "not moving forward" notices
- Assessments or coding challenges sent by a company

It is NOT a job email if it is:
- A newsletter, digest, or promotional email
- A job alert or recommendation (e.g. "Jobs you might like")
- A LinkedIn notification, social update, or Quora digest
- A general company marketing email
- Anything not directly about a specific application you submitted

Return ONLY a JSON object, no extra text:
{
    "relevant": true,
    "company": "Company Name",
    "title": "Job Title",
    "status": "Applied"
}

If not relevant, return:
{
    "relevant": false,
    "company": "",
    "title": "",
    "status": ""
}

Status must be one of: "Applied", "Interview", "Offer", "Rejected"
- "Applied" — confirmation, received, thank you for applying
- "Interview" — invited to interview, phone screen, assessment, scheduling
- "Offer" — offer letter, congratulations on your offer
- "Rejected" — not moving forward, regret to inform, other candidates chosen

Email Subject: ` + subject + `

Email Body: ` + body

	req := openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a job application email classifier and parser. Return JSON only.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature: 0.1,
	}

	var resp openai.ChatCompletionResponse
	var err error
	backoff := 5 * time.Second
	for attempt := 0; attempt < 5; attempt++ {
		resp, err = client.CreateChatCompletion(ctx, req)
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), "429") {
			LogError(fmt.Sprintf("Rate limited, retrying in %s...", backoff))
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		return "", "", "", false, fmt.Errorf("OpenAI API error: %v", err)
	}
	if err != nil {
		return "", "", "", false, fmt.Errorf("OpenAI API error after retries: %v", err)
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		return "", "", "", false, errors.New("no JSON found in OpenAI response")
	}

	var result struct {
		Relevant bool   `json:"relevant"`
		Company  string `json:"company"`
		Title    string `json:"title"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal([]byte(content[jsonStart:jsonEnd+1]), &result); err != nil {
		return "", "", "", false, fmt.Errorf("failed to parse JSON: %v", err)
	}

	if !result.Relevant {
		return "", "", "", false, nil
	}

	if result.Company == "" || result.Title == "" {
		return "", "", "", false, errors.New("missing company or title in extracted data")
	}

	switch result.Status {
	case "Interview", "Offer", "Rejected":
	default:
		result.Status = "Applied"
	}

	return result.Company, result.Title, result.Status, true, nil
}

func TestOpenAI() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not set")
		return
	}

	client := openai.NewClient(apiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o, // or openai.GPT4o
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: "Say hello!",
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("OpenAI API error: %v\n", err)
		return
	}

	fmt.Println("OpenAI response:", resp.Choices[0].Message.Content)
}
