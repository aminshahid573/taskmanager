package worker

import (
	"bytes"
	"fmt"
)

func (w *EmailWorker) buildTaskAssignedEmail(job EmailJob) (string, string) {
	subject := fmt.Sprintf("New Task Assigned: %s", job.TaskTitle)

	data := struct {
		EmailType       string
		RecipientName   string
		TaskTitle       string
		OrgName         string
		DueDate         string
		ExtraNote       string
		ActionURL       string
		BackgroundColor string
		PrimaryColor    string
	}{
		EmailType:       "task_assigned",
		RecipientName:   job.RecipientName,
		TaskTitle:       job.TaskTitle,
		OrgName:         job.OrgName,
		DueDate:         formatDueDate(job.DueDate),
		ExtraNote:       job.ExtraNote,
		ActionURL:       job.ActionURL,
		BackgroundColor: "#f8fafc",
		PrimaryColor:    "#2563eb",
	}

	var body bytes.Buffer
	if err := w.templates.ExecuteTemplate(&body, "base", data); err != nil {
		panic(err)
	}

	return subject, body.String()
}

func (w *EmailWorker) buildDueSoonEmail(job EmailJob) (string, string) {
	subject := fmt.Sprintf("Task Due Soon: %s", job.TaskTitle)

	data := struct {
		EmailType       string
		RecipientName   string
		TaskTitle       string
		OrgName         string
		DueDate         string
		ActionURL       string
		BackgroundColor string
		PrimaryColor    string
	}{
		EmailType:       "due_soon",
		RecipientName:   job.RecipientName,
		TaskTitle:       job.TaskTitle,
		OrgName:         job.OrgName,
		DueDate:         formatDueDate(job.DueDate),
		ActionURL:       job.ActionURL,
		BackgroundColor: "#f8fafc",
		PrimaryColor:    "#f59e0b",
	}

	var body bytes.Buffer
	if err := w.templates.ExecuteTemplate(&body, "base", data); err != nil {
		panic(err)
	}

	return subject, body.String()
}

func (w *EmailWorker) buildOverdueEmail(job EmailJob) (string, string) {
	subject := fmt.Sprintf("Task Overdue: %s", job.TaskTitle)

	data := struct {
		EmailType       string
		RecipientName   string
		TaskTitle       string
		OrgName         string
		DueDate         string
		ActionURL       string
		BackgroundColor string
		PrimaryColor    string
	}{
		EmailType:       "overdue",
		RecipientName:   job.RecipientName,
		TaskTitle:       job.TaskTitle,
		OrgName:         job.OrgName,
		DueDate:         formatDueDate(job.DueDate),
		ActionURL:       job.ActionURL,
		BackgroundColor: "#f8fafc",
		PrimaryColor:    "#dc2626",
	}

	var body bytes.Buffer
	if err := w.templates.ExecuteTemplate(&body, "base", data); err != nil {
		panic(err)
	}

	return subject, body.String()
}

func (w *EmailWorker) buildOTPEmail(job EmailJob) (string, string) {
	subject := "Verify Your Email - OTP Code"

	data := struct {
		EmailType       string
		RecipientName   string
		OTPCode         string
		BackgroundColor string
		PrimaryColor    string
	}{
		EmailType:       "otp_verification",
		RecipientName:   job.RecipientName,
		OTPCode:         job.OTPCode,
		BackgroundColor: "#f8fafc",
		PrimaryColor:    "#2563eb",
	}

	var body bytes.Buffer
	if err := w.templates.ExecuteTemplate(&body, "base", data); err != nil {
		panic(err)
	}

	return subject, body.String()
}

