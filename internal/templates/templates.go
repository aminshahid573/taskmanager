package templates

import (
	"embed"
	"html/template"
)

//go:embed email/*.html
var emailTemplatesFS embed.FS

func LoadEmailTemplates() (*template.Template, error) {
	return template.ParseFS(
		emailTemplatesFS,
		"email/base.html",
		"email/otp.html",
		"email/overdue.html",
		"email/due_soon.html",
		"email/task_assigned.html",
	)
}

