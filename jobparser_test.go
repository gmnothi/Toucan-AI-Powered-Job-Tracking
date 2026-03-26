package main

import (
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
}

var testEmails = []struct {
	name            string
	subject         string
	body            string
	wantRelevant    bool
	wantStatus      string
}{
	{
		name:         "application confirmation",
		subject:      "Thank you – we've received your Tesla application",
		body:         "Hi Gurman, thank you for applying to the Software Engineering Intern position at Tesla. We have received your application and will review it shortly.",
		wantRelevant: true,
		wantStatus:   "Applied",
	},
	{
		name:         "interview invite",
		subject:      "Interview Invitation – Google Software Engineer",
		body:         "We'd like to invite you to interview for the Software Engineer role. Please use the link below to schedule a 45-minute phone screen.",
		wantRelevant: true,
		wantStatus:   "Interview",
	},
	{
		name:         "rejection",
		subject:      "Update on your application to Stripe",
		body:         "Hi Gurman, thank you for applying to the Backend Engineer position at Stripe. After careful consideration, we have decided not to move forward with your application at this time. We appreciate your interest and wish you the best in your job search.",
		wantRelevant: true,
		wantStatus:   "Rejected",
	},
	{
		name:         "offer letter",
		subject:      "Congratulations! Your offer from Shopify",
		body:         "We are thrilled to extend an offer of employment for the position of Backend Engineer at Shopify. Please find your offer letter attached.",
		wantRelevant: true,
		wantStatus:   "Offer",
	},
	{
		name:         "job alert digest — should be ignored",
		subject:      "10 new jobs matching Software Engineer in Toronto",
		body:         "Based on your profile, here are jobs you might like: Software Engineer at Acme, Backend Dev at Foo...",
		wantRelevant: false,
	},
	{
		name:         "quora digest — should be ignored",
		subject:      "Top stories on Quora this week",
		body:         "Here are the top answers you might have missed: What is the best programming language?",
		wantRelevant: false,
	},
	{
		name:         "marketing email — should be ignored",
		subject:      "Exclusive deals just for you",
		body:         "Check out our latest offers and discounts on products you love.",
		wantRelevant: false,
	},
}

func TestExtractJobDetails(t *testing.T) {
	for _, tt := range testEmails {
		t.Run(tt.name, func(t *testing.T) {
			company, title, status, relevant, err := ExtractJobDetails(tt.subject, tt.body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if relevant != tt.wantRelevant {
				t.Errorf("relevant = %v, want %v", relevant, tt.wantRelevant)
			}

			if !tt.wantRelevant {
				return
			}

			if company == "" {
				t.Error("company is empty")
			}
			if title == "" {
				t.Error("title is empty")
			}
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}

			t.Logf("company=%q title=%q status=%q", company, title, status)
		})
	}
}
