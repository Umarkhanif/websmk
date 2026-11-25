package model

import "time"

// SessionInfo describes where a session was created. It lives beside the Redis
// session token so the account security page can show the active device.
type SessionInfo struct {
	UserID    string    `json:"userId"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
}

// LoginEvent is one entry in the append-only login audit trail.
type LoginEvent struct {
	ID         int64
	UserID     string
	Identifier string
	Role       string
	Method     string
	Success    bool
	Reason     string
	IP         string
	UserAgent  string
	CreatedAt  time.Time
}
