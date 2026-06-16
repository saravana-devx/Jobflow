// File: internal/email/dto.go
package email

// SendEmailRequest is the input for sending a transactional email.
type SendEmailRequest struct {
	To      string `json:"to" binding:"required,email"`
	Subject string `json:"subject" binding:"required"`
	Body    string `json:"body" binding:"required"`
}

// SendEmailResult is returned once an email has been accepted for delivery.
type SendEmailResult struct {
	MessageID string `json:"messageId"`
}
