// Package android is reserved for Android cloud-build adaptation (T03).
package android

import "fmt"

// ErrNotImplemented is returned until T03 is scheduled.
var ErrNotImplemented = fmt.Errorf("android cloud build is reserved (T03)")

// Submit is a stub so callers can branch on platform without missing packages.
func Submit(_ any) error {
	return ErrNotImplemented
}
