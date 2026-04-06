package notification

import "errors"

// ErrAdapterNotFound is returned when no adapter is registered for a channel type.
var ErrAdapterNotFound = errors.New("notification adapter not found")
