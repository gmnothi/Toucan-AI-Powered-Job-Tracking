package main

import (
	"context"
	"encoding/base64"
	"log"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func ScanGmailForUser(userID int, since time.Time) error {
	refreshToken, err := GetUserRefreshToken(userID)
	if err != nil || refreshToken == "" {
		return err
	}

	cfg := GetOAuthConfig()
	tokenSource := cfg.TokenSource(context.Background(), &oauth2.Token{
		RefreshToken: refreshToken,
	})

	svc, err := gmail.NewService(context.Background(), option.WithTokenSource(tokenSource))
	if err != nil {
		return err
	}

	query := "in:inbox"
	if !since.IsZero() {
		query += " after:" + since.Format("2006/01/02")
	} else {
		// Default: last 90 days
		query += " after:" + time.Now().AddDate(0, -3, 0).Format("2006/01/02")
	}

	Progress.mu.Lock()
	Progress.Running = true
	Progress.Paused = false
	Progress.Total = 0
	Progress.Processed = 0
	Progress.Saved = 0
	Progress.Account = "Gmail"
	Progress.mu.Unlock()

	var pageToken string
	var allIDs []string

	for {
		call := svc.Users.Messages.List("me").Q(query).MaxResults(100)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		res, err := call.Do()
		if err != nil {
			log.Printf("[Gmail] list error: %v", err)
			break
		}
		for _, m := range res.Messages {
			allIDs = append(allIDs, m.Id)
		}
		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}

	Progress.mu.Lock()
	Progress.Total = len(allIDs)
	Progress.mu.Unlock()

	for _, msgID := range allIDs {
		waitIfPaused()

		msg, err := svc.Users.Messages.Get("me", msgID).Format("full").Do()
		if err != nil {
			continue
		}

		Progress.mu.Lock()
		Progress.Processed++
		Progress.mu.Unlock()

		subject, from, dateStr := extractGmailHeaders(msg)
		if !isCareerDomain(from) {
			continue
		}
		if JobExists(msgID, userID) {
			continue
		}

		body := extractGmailBody(msg)
		time.Sleep(500 * time.Millisecond)

		company, title, status, relevant, err := ExtractJobDetails(subject, body)
		if err != nil {
			log.Printf("[Gmail] GPT error: %v", err)
			continue
		}
		if !relevant {
			LogSkipped(subject)
			continue
		}
		LogFlagged(company, title, status)

		if existing := FindJobByCompanyTitle(company, title, userID); existing != nil {
			UpdateJobStatusAndEmail(existing.ID, status, msgID, subject, body, userID)
			LogFlagged(company, title, status+" (updated)")
		} else {
			SaveJob(Job{
				Company: company,
				Title:   title,
				Status:  status,
				EmailID: msgID,
				Date:    dateStr,
				Subject: subject,
				Body:    body,
			}, userID)
		}

		Progress.mu.Lock()
		Progress.Saved++
		Progress.mu.Unlock()
	}

	Progress.mu.Lock()
	Progress.Running = false
	Progress.mu.Unlock()

	return nil
}

func extractGmailHeaders(msg *gmail.Message) (subject, from, date string) {
	for _, h := range msg.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "subject":
			subject = h.Value
		case "from":
			from = h.Value
		case "date":
			t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", h.Value)
			if err != nil {
				t, err = time.Parse("Mon, 2 Jan 2006 15:04:05 -0700 (MST)", h.Value)
			}
			if err == nil {
				date = t.Format("2006-01-02")
			}
		}
	}
	return
}

func extractGmailBody(msg *gmail.Message) string {
	return extractPartBody(msg.Payload)
}

func extractPartBody(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}
	if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			return string(data)
		}
	}
	for _, p := range part.Parts {
		if body := extractPartBody(p); body != "" {
			return body
		}
	}
	return ""
}
