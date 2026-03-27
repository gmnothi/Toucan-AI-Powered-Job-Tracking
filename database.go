package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Cannot connect to database:", err)
	}

	db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id            SERIAL PRIMARY KEY,
			google_id     TEXT UNIQUE NOT NULL,
			email         TEXT NOT NULL,
			refresh_token TEXT,
			created_at    TIMESTAMPTZ DEFAULT NOW()
		)`)
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS refresh_token TEXT`)

	db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id       SERIAL PRIMARY KEY,
			company  TEXT,
			title    TEXT,
			status   TEXT,
			email_id TEXT UNIQUE,
			date     TEXT,
			subject  TEXT,
			body     TEXT,
			user_id  INTEGER REFERENCES users(id)
		)`)

	db.Exec(`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id)`)
}

func UpsertUser(googleID, email, refreshToken string) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO users (google_id, email, refresh_token)
		VALUES ($1, $2, $3)
		ON CONFLICT (google_id) DO UPDATE
		  SET email = EXCLUDED.email,
		      refresh_token = CASE WHEN EXCLUDED.refresh_token != '' THEN EXCLUDED.refresh_token ELSE users.refresh_token END
		RETURNING id`, googleID, email, refreshToken).Scan(&id)
	return id, err
}

func GetUserRefreshToken(userID int) (string, error) {
	var token string
	err := db.QueryRow("SELECT COALESCE(refresh_token,'') FROM users WHERE id = $1", userID).Scan(&token)
	return token, err
}

func GetUserEmail(userID int) (string, error) {
	var email string
	err := db.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	return email, err
}

func SaveJob(job Job, userID int) {
	stmt, _ := db.Prepare(`
		INSERT INTO jobs(company, title, status, email_id, date, subject, body, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email_id) DO NOTHING`)
	_, err := stmt.Exec(job.Company, job.Title, job.Status, job.EmailID, job.Date, job.Subject, job.Body, userID)
	if err != nil {
		log.Println("Failed to insert job:", err)
	}
}

func GetJobsForUser(userID int) []Job {
	rows, _ := db.Query(`
		SELECT id, company, title, status, email_id, date, COALESCE(subject,''), COALESCE(body,'')
		FROM jobs WHERE user_id = $1`, userID)
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var job Job
		rows.Scan(&job.ID, &job.Company, &job.Title, &job.Status, &job.EmailID, &job.Date, &job.Subject, &job.Body)
		jobs = append(jobs, job)
	}
	return jobs
}

func ClearJobs() {
	_, err := db.Exec("DELETE FROM jobs")
	if err != nil {
		log.Println("Failed to clear jobs table:", err)
	} else {
		log.Println("Cleared old jobs from DB")
	}
}

func JobExists(emailID string, userID int) bool {
	var count int
	db.QueryRow("SELECT COUNT(1) FROM jobs WHERE email_id = $1 AND user_id = $2", emailID, userID).Scan(&count)
	return count > 0
}

func FindJobByCompanyTitle(company, title string, userID int) *Job {
	var job Job
	err := db.QueryRow(`
		SELECT id, company, title, status, email_id, date, COALESCE(subject,''), COALESCE(body,'')
		FROM jobs WHERE LOWER(company)=LOWER($1) AND LOWER(title)=LOWER($2) AND user_id = $3`,
		company, title, userID).Scan(&job.ID, &job.Company, &job.Title, &job.Status, &job.EmailID, &job.Date, &job.Subject, &job.Body)
	if err != nil {
		return nil
	}
	return &job
}

func UpdateJobStatusAndEmail(id int, status, emailID, subject, body string, userID int) {
	db.Exec("UPDATE jobs SET status=$1, email_id=$2, subject=$3, body=$4 WHERE id=$5 AND user_id=$6",
		status, emailID, subject, body, id, userID)
}

func DeleteJob(id string, userID int) error {
	_, err := db.Exec("DELETE FROM jobs WHERE id = $1 AND user_id = $2", id, userID)
	return err
}

func UpdateJobStatus(id string, status string, userID int) error {
	_, err := db.Exec("UPDATE jobs SET status = $1 WHERE id = $2 AND user_id = $3", status, id, userID)
	return err
}
