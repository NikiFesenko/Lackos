package api

// Standard error codes returned in ErrorResponse.Code.
// Clients can switch on these codes without parsing the message string.
const (
	ErrCodeUnauthorized   = "unauthorized"
	ErrCodeForbidden      = "forbidden"
	ErrCodeNotFound       = "not_found"
	ErrCodeBadRequest     = "bad_request"
	ErrCodeConflict       = "conflict"
	ErrCodeInternal       = "internal_error"
	ErrCodeInvalidUUID    = "invalid_uuid"
	ErrCodeInvalidPayload = "invalid_payload"
)
