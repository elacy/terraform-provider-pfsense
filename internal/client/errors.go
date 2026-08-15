package client

import "errors"

// ErrNotFound is returned by Get when the requested object does not exist.
// Resources use it to distinguish "absent" (needs create) from "present".
var ErrNotFound = errors.New("pfSense object not found")
