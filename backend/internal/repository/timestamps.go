package repository

import "time"

func ensureTimestamps(createdAt *time.Time, updatedAt *time.Time) {
	now := time.Now().UTC()
	if createdAt != nil && createdAt.IsZero() {
		*createdAt = now
	}
	if updatedAt != nil && updatedAt.IsZero() {
		*updatedAt = now
	}
}
