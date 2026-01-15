package worker

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log/slog"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/google/uuid"
	"github.com/aminshahid573/taskmanager/internal/config"
	"github.com/aminshahid573/taskmanager/internal/templates"
)

type EmailJob struct {
	Type           string // "task_assigned", "due_soon", "overdue"
	RecipientEmail string
	RecipientName  string
	TaskID         uuid.UUID
	TaskTitle      string
	OrgID          uuid.UUID
	OrgName        string
	DueDate        *time.Time
	OTPCode        string
	ActionURL      string
	ExtraNote      string
}

type EmailWorker struct {
	cfg       config.EmailConfig
	logger    *slog.Logger
	jobs      chan EmailJob
	templates *template.Template
}

func NewEmailWorker(cfg config.EmailConfig, logger *slog.Logger) (*EmailWorker, error) {
	tmpl, err := templates.LoadEmailTemplates()
	if err != nil {
		return nil, err
	}
	return &EmailWorker{
		cfg:       cfg,
		logger:    logger,
		jobs:      make(chan EmailJob, 100), // Buffer of 100 jobs
		templates: tmpl,
	}, nil
}

func (w *EmailWorker) Start(ctx context.Context) {
	w.logger.Info("Email worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Email worker stopping")
			close(w.jobs)
			return
		case job := <-w.jobs:
			if err := w.ProcessJob(job); err != nil {
				w.logger.Error("Failed to process email job",
					"error", err,
					"type", job.Type,
					"task_id", job.TaskID,
				)
			} else {
				w.logger.Info("Email sent successfully",
					"type", job.Type,
					"task_id", job.TaskID,
					"recipient", job.RecipientEmail,
				)
			}
		}
	}
}

func (w *EmailWorker) QueueJob(job EmailJob) {
	select {
	case w.jobs <- job:
		w.logger.Debug("Email job queued", "type", job.Type, "task_id", job.TaskID)
	default:
		w.logger.Warn("Email job queue full, dropping job", "type", job.Type)
	}
}

func (w *EmailWorker) ProcessJob(job EmailJob) error {
	// Validate job has required fields
	if job.RecipientEmail == "" {
		return fmt.Errorf("recipient email is required for job type: %s", job.Type)
	}

	var subject, body string

	switch job.Type {
	case "task_assigned":
		subject, body = w.buildTaskAssignedEmail(job)
	case "due_soon":
		subject, body = w.buildDueSoonEmail(job)
	case "overdue":
		subject, body = w.buildOverdueEmail(job)
	case "otp_verification":
		subject, body = w.buildOTPEmail(job)
	default:
		return fmt.Errorf("unknown email type: %s", job.Type)
	}

	return w.sendEmail(job.RecipientEmail, subject, body)
}

func (w *EmailWorker) sendEmail(to, subject, body string) error {
	// Skip sending if SMTP is not configured (development mode)
	if w.cfg.SMTPHost == "" || w.cfg.SMTPHost == "smtp.example.com" {
		w.logger.Info("SMTP not configured, skipping email send", "to", to, "subject", subject)
		return nil
	}

	// Validate recipient email
	if to == "" {
		return fmt.Errorf("recipient email is empty")
	}

	// Create email message properly using net/mail
	from := mail.Address{Name: w.cfg.FromName, Address: w.cfg.FromEmail}
	toAddr := mail.Address{Address: to}

	// Build headers with proper line endings
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from.String()))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", toAddr.String()))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("\r\n")

	// Write body as-is for HTML emails
	msg.WriteString(body)

	auth := smtp.PlainAuth("", w.cfg.SMTPUsername, w.cfg.SMTPPassword, w.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", w.cfg.SMTPHost, w.cfg.SMTPPort)

	// Use TLS if port is 465, otherwise use STARTTLS
	if w.cfg.SMTPPort == 465 {
		return w.sendEmailTLS(addr, auth, w.cfg.FromEmail, []string{to}, msg.Bytes())
	}

	return smtp.SendMail(addr, auth, w.cfg.FromEmail, []string{to}, msg.Bytes())
}

func (w *EmailWorker) sendEmailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: w.cfg.SMTPHost,
	}

	// Connect to server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, w.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	defer client.Close()

	// Auth
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail: %w", err)
	}

	// Set recipients
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("rcpt: %w", err)
		}
	}

	// Send data
	dataWriter, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}

	_, err = dataWriter.Write(msg)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if err := dataWriter.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return client.Quit()
}

func formatDueDate(dueDate *time.Time) string {
	if dueDate == nil {
		return "Due Date: Not set"
	}
	return fmt.Sprintf("Due Date: %s", dueDate.Format("2006-01-02 15:04"))
}

