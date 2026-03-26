package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB() {
	var err error
	db, err = sql.Open("sqlite3", "./jobs.db")
	if err != nil {
		log.Fatal(err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		company TEXT,
		title TEXT,
		status TEXT,
		email_id TEXT UNIQUE,
		date TEXT,
		subject TEXT,
		body TEXT
	);`
	// Add columns if upgrading from older schema
	db.Exec(`ALTER TABLE jobs ADD COLUMN subject TEXT`)
	db.Exec(`ALTER TABLE jobs ADD COLUMN body TEXT`)
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}
}

func SaveJob(job Job) {
	stmt, _ := db.Prepare("INSERT OR IGNORE INTO jobs(company, title, status, email_id, date, subject, body) VALUES (?, ?, ?, ?, ?, ?, ?)")
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
	db, err := sql.Open("sqlite3", "./jobs.db")
	if err != nil {
		log.Println("Failed to open DB:", err)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM jobs")
	if err != nil {
		log.Println("Failed to clear jobs table:", err)
	} else {
		log.Println("Cleared old jobs from DB")
	}
}

func JobExists(emailID string) bool {
	var count int
	db.QueryRow("SELECT COUNT(1) FROM jobs WHERE email_id = ?", emailID).Scan(&count)
	return count > 0
}

func FindJobByCompanyTitle(company, title string) *Job {
	var job Job
	err := db.QueryRow(
		"SELECT id, company, title, status, email_id, date, COALESCE(subject,''), COALESCE(body,'') FROM jobs WHERE LOWER(company)=LOWER(?) AND LOWER(title)=LOWER(?)",
		company, title,
	).Scan(&job.ID, &job.Company, &job.Title, &job.Status, &job.EmailID, &job.Date, &job.Subject, &job.Body)
	if err != nil {
		return nil
	}
	return &job
}

func UpdateJobStatusAndEmail(id int, status, emailID, subject, body string) {
	db.Exec("UPDATE jobs SET status=?, email_id=?, subject=?, body=? WHERE id=?", status, emailID, subject, body, id)
}

func DeleteJob(id string) error {
	query := `DELETE FROM jobs WHERE ID = ?`
	_, err := db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func UpdateJobStatus(id string, status string) error {
	query := `UPDATE jobs SET status = ? WHERE id = ?`
	_, err := db.Exec(query, status, id)
	return err
}
