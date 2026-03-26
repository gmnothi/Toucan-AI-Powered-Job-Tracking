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

	createTable := `
	CREATE TABLE IF NOT EXISTS jobs (
		id SERIAL PRIMARY KEY,
		company TEXT,
		title TEXT,
		status TEXT,
		email_id TEXT UNIQUE,
		date TEXT,
		subject TEXT,
		body TEXT
	);`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}
}

func SaveJob(job Job) {
	stmt, _ := db.Prepare("INSERT INTO jobs(company, title, status, email_id, date, subject, body) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (email_id) DO NOTHING")
	_, err := stmt.Exec(job.Company, job.Title, job.Status, job.EmailID, job.Date, job.Subject, job.Body)
	if err != nil {
		log.Println("Failed to insert job:", err)
	}
}

func GetAllJobs() []Job {
	rows, _ := db.Query("SELECT id, company, title, status, email_id, date, COALESCE(subject,''), COALESCE(body,'') FROM jobs")
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

func JobExists(emailID string) bool {
	var count int
	db.QueryRow("SELECT COUNT(1) FROM jobs WHERE email_id = $1", emailID).Scan(&count)
	return count > 0
}

func FindJobByCompanyTitle(company, title string) *Job {
	var job Job
	err := db.QueryRow(
		"SELECT id, company, title, status, email_id, date, COALESCE(subject,''), COALESCE(body,'') FROM jobs WHERE LOWER(company)=LOWER($1) AND LOWER(title)=LOWER($2)",
		company, title,
	).Scan(&job.ID, &job.Company, &job.Title, &job.Status, &job.EmailID, &job.Date, &job.Subject, &job.Body)
	if err != nil {
		return nil
	}
	return &job
}

func UpdateJobStatusAndEmail(id int, status, emailID, subject, body string) {
	db.Exec("UPDATE jobs SET status=$1, email_id=$2, subject=$3, body=$4 WHERE id=$5", status, emailID, subject, body, id)
}

func DeleteJob(id string) error {
	_, err := db.Exec("DELETE FROM jobs WHERE id = $1", id)
	return err
}

func UpdateJobStatus(id string, status string) error {
	_, err := db.Exec("UPDATE jobs SET status = $1 WHERE id = $2", status, id)
	return err
}
