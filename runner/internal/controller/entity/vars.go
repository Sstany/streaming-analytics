package entity

type Status int

const (
	Init Status = iota
	Created
	InProgress
	Done
)

type Job struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
	URL    string `json:"url" binding:"required"`
}
