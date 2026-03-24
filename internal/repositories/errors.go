package repositories

import "errors"

// Repository errors
var (
	ErrDeviceNotFound = errors.New("device_not_found")
	ErrGroupNotFound  = errors.New("group_not_found")
	ErrTagNotFound    = errors.New("tag_not_found")
	ErrEventNotFound  = errors.New("event_not_found")
)
