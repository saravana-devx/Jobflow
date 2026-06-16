// File: internal/email/service.go
package email

import "context"

// Sender is the email-provider abstraction. Concrete implementations (SMTP,
// SES, Resend, etc.) satisfy this interface so callers don't depend on a
// specific provider.
type Sender interface {
	Send(ctx context.Context, req *SendEmailRequest) (*SendEmailResult, error)
}

// Service is the application-facing email service.
type Service struct {
	// provider config (SMTP host, API key, ...) goes here once a provider is chosen
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Send(ctx context.Context, req *SendEmailRequest) (*SendEmailResult, error) {
	// TODO: implement by developer
	// Hint: hand the request to a real email provider and return its message ID.
	panic("not implemented")
}
