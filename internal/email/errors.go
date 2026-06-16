// File: internal/email/errors.go
package email

import "errors"

var (
	ErrInvalidRecipient = errors.New("invalid recipient address")
	ErrSendFailed       = errors.New("failed to send email")
)
