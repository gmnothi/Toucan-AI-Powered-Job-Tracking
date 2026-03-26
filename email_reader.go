package main

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	imapmail "github.com/emersion/go-message/mail"
	"golang.org/x/net/html/charset"
)

type ScanProgress struct {
	mu        sync.RWMutex
	Running   bool   `json:"running"`
	Paused    bool   `json:"paused"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Saved     int    `json:"saved"`
	Account   string `json:"account"`
}

var Progress = &ScanProgress{}

func waitIfPaused() {
	for {
		Progress.mu.RLock()
		paused := Progress.Paused
		Progress.mu.RUnlock()
		if !paused {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func init() {
	message.CharsetReader = func(charsetStr string, input io.Reader) (io.Reader, error) {
		return charset.NewReaderLabel(charsetStr, input)
	}
}

type inboxAccount struct {
	host  string
	user  string
	pass  string
	label string
}

func getConfiguredAccounts() []inboxAccount {
	var accounts []inboxAccount

	if u, p := os.Getenv("GMAIL_USER"), os.Getenv("GMAIL_PASS"); u != "" && p != "" {
		accounts = append(accounts, inboxAccount{
			host:  "imap.gmail.com:993",
			user:  u,
			pass:  p,
			label: "Gmail",
		})
	}

	if u, p := os.Getenv("OUTLOOK_USER"), os.Getenv("OUTLOOK_PASS"); u != "" && p != "" {
		accounts = append(accounts, inboxAccount{
			host:  "outlook.office365.com:993",
			user:  u,
			pass:  p,
			label: "Outlook",
		})
	}

	return accounts
}

func CheckInbox() {
	CheckInboxSince(time.Time{})
}

func CheckInboxSince(since time.Time) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in CheckInboxSince: %v", r)
		}
	}()

	accounts := getConfiguredAccounts()
	if len(accounts) == 0 {
		log.Println("No inbox accounts configured")
		return
	}

	Progress.mu.Lock()
	Progress.Running = true
	Progress.Paused = false
	Progress.Total = 0
	Progress.Processed = 0
	Progress.Saved = 0
	Progress.Account = ""
	Progress.mu.Unlock()

	for _, acc := range accounts {
		if since.IsZero() {
			msg := "Checking " + acc.label + " inbox (" + acc.user + ")..."
			log.Println(msg)
			LogInfo(msg)
		} else {
			msg := "Checking " + acc.label + " inbox (" + acc.user + ") since " + since.Format("2006-01-02") + "..."
			log.Println(msg)
			LogInfo(msg)
		}
		checkInboxForAccount(acc, since)
	}

	Progress.mu.Lock()
	Progress.Running = false
	Progress.mu.Unlock()
}

func checkInboxForAccount(acc inboxAccount, since time.Time) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in checkInboxForAccount (%s): %v", acc.label, r)
		}
	}()

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", acc.host, &tls.Config{})
	if err != nil {
		log.Printf("[%s] IMAP dial error: %v", acc.label, err)
		return
	}

	c, err := client.New(conn)
	if err != nil {
		log.Printf("[%s] IMAP client creation failed: %v", acc.label, err)
		return
	}
	defer c.Logout()

	err = c.Login(acc.user, acc.pass)
	if err != nil {
		log.Printf("[%s] Login failed: %v", acc.label, err)
		return
	}

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Printf("[%s] Unable to select inbox: %v", acc.label, err)
		return
	}

	if mbox.Messages == 0 {
		return
	}

	// Show inbox size immediately so the UI has a number while searching
	Progress.mu.Lock()
	Progress.Account = acc.label
	Progress.Total += int(mbox.Messages)
	Progress.mu.Unlock()

	seqSet := new(imap.SeqSet)
	var totalToFetch int
	if !since.IsZero() {
		criteria := imap.NewSearchCriteria()
		criteria.Since = since
		ids, err := c.Search(criteria)
		if err != nil {
			log.Printf("[%s] Search failed: %v", acc.label, err)
			return
		}
		if len(ids) == 0 {
			log.Printf("[%s] No emails found since %s", acc.label, since.Format("2006-01-02"))
			return
		}
		log.Printf("[%s] Found %d emails since %s", acc.label, len(ids), since.Format("2006-01-02"))
		seqSet.AddNum(ids...)
		totalToFetch = len(ids)
		// Update total to the actual filtered count
		Progress.mu.Lock()
		Progress.Total = Progress.Total - int(mbox.Messages) + totalToFetch
		Progress.mu.Unlock()
	} else {
		const fetchCount = 1000
		from := uint32(1)
		if mbox.Messages > fetchCount {
			from = mbox.Messages - fetchCount + 1
		}
		seqSet.AddRange(from, mbox.Messages)
		totalToFetch = int(mbox.Messages-from) + 1
		// Update total to actual fetch count
		Progress.mu.Lock()
		Progress.Total = Progress.Total - int(mbox.Messages) + totalToFetch
		Progress.mu.Unlock()
	}

	section := &imap.BodySectionName{}
	messages := make(chan *imap.Message, mbox.Messages+1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[%s] Panic recovered in fetch goroutine: %v", acc.label, r)
			}
		}()

		if err := c.Fetch(seqSet, []imap.FetchItem{
			section.FetchItem(),
			imap.FetchEnvelope,
		}, messages); err != nil {
			log.Printf("[%s] Fetch failed: %v", acc.label, err)
		}
	}()

	emailCount := 0
	for msg := range messages {
		waitIfPaused()
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[%s] Panic processing message: %v", acc.label, r)
				}
			}()

			emailCount++
			Progress.mu.Lock()
			Progress.Processed++
			Progress.mu.Unlock()

			if msg == nil || msg.Envelope == nil {
				return
			}

			r := msg.GetBody(section)
			if r == nil {
				return
			}

			mr, err := imapmail.CreateReader(r)
			if err != nil {
				log.Printf("[%s] Could not parse message: %v", acc.label, err)
				return
			}

			header := mr.Header
			subject, _ := header.Subject()
			fromList, _ := header.AddressList("From")
			fromAddress := "unknown"
			if len(fromList) > 0 && fromList[0] != nil && fromList[0].Address != "" {
				fromAddress = fromList[0].Address
			}

			LogReading(subject, fromAddress)

			if isCareerDomain(fromAddress) {
				messageID := msg.Envelope.MessageId

				if JobExists(messageID) {
					return
				}

				dateStr := "unknown"
				if !msg.Envelope.Date.IsZero() {
					dateStr = msg.Envelope.Date.Format("2006-01-02")
				}

				body := ExtractPlainTextBody(mr)
				time.Sleep(500 * time.Millisecond) // stay under TPM limit
				company, title, status, relevant, err := ExtractJobDetails(subject, body)
				if err != nil {
					log.Printf("[%s] GPT error: %v", acc.label, err)
					LogError("GPT error: " + err.Error())
					return
				}
				if !relevant {
					LogSkipped(subject)
					return
				}
				LogFlagged(company, title, status)

				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("[%s] Panic in SaveJob: %v", acc.label, r)
						}
					}()
					if existing := FindJobByCompanyTitle(company, title); existing != nil {
						UpdateJobStatusAndEmail(existing.ID, status, messageID, subject, body)
						LogFlagged(company, title, status+" (updated)")
					} else {
						SaveJob(Job{
							Company: company,
							Title:   title,
							Status:  status,
							EmailID: messageID,
							Date:    dateStr,
							Subject: subject,
							Body:    body,
						})
					}
					Progress.mu.Lock()
					Progress.Saved++
					Progress.mu.Unlock()
				}()
			}
		}()
	}

	log.Printf("[%s] Processed %d emails", acc.label, emailCount)
}

func isCareerDomain(address string) bool {
	address = strings.ToLower(address)

	excluded := []string{
		"@linkedin.com",
		"@quora.com",
		"@indeed.com",
	}
	for _, ex := range excluded {
		if strings.Contains(address, ex) {
			return false
		}
	}


	domains := []string{
		"indeed.com",
		"workdaymail.com",
		"jobs.noreply@",
		"myworkdayjobs.com",
		"glassdoor.com",
		"jobvite.com",
		"lever.co",
		"greenhouse.io",
		"careers@",
		"no-reply",
		"autoreply",
		"noreply",
		"reply@",
		"do-not-reply@",
		"workday",
		"icims",
		"talent",
		// Company-specific
		"tesla.com",
		// Personal forwarding
		"othigurman@gmail.com",
	}

	for _, domain := range domains {
		if strings.Contains(address, domain) {
			return true
		}
	}

	return false
}


func ExtractPlainTextBody(mr *imapmail.Reader) string {
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Error reading MIME part:", err)
			break
		}

		switch h := p.Header.(type) {
		case *imapmail.InlineHeader:
			mediaType, _, _ := h.ContentType()
			if strings.HasPrefix(mediaType, "text/plain") {
				bodyBytes, err := io.ReadAll(p.Body)
				if err != nil {
					log.Println("Failed to read plain body:", err)
					return ""
				}
				return string(bodyBytes)
			}
		}
	}
	return ""
}

