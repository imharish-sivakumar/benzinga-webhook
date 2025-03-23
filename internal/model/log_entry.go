// Package model contains data structures used across the webhook application.
package model

// Login represents a login event with a timestamp and originating IP address.
type Login struct {
	Time string `json:"time" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	IP   string `json:"ip" validate:"required,ip"`
}

// PhoneNumbers holds the home and mobile phone numbers of a user.
type PhoneNumbers struct {
	Home   string `json:"home" validate:"required,phoneformat"`
	Mobile string `json:"mobile" validate:"required,phoneformat"`
}

// Meta includes additional metadata about the user such as login history and phone numbers.
type Meta struct {
	Logins       []Login      `json:"logins" validate:"required,dive"`
	PhoneNumbers PhoneNumbers `json:"phone_numbers"`
}

// LogEntry represents the structure of incoming JSON payloads from the webhook.
type LogEntry struct {
	UserID    int     `json:"user_id" validate:"required,gte=1"`
	Total     float64 `json:"total" validate:"required,gt=0"`
	Title     string  `json:"title" validate:"required,min=3"`
	Meta      Meta    `json:"meta" validate:"required"`
	Completed bool    `json:"completed"`
}
