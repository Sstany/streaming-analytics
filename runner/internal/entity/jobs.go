package entity

type Status string

const (
	Undefined          Status = "UNDEFINED"
	Init               Status = "INIT"
	Startup            Status = "STARTUP"
	Active             Status = "ACTIVE"
	Shutdown           Status = "SHUTDOWN"
	ShutdownProcessing Status = "SHUTDOWN_PROCESSING"
	Inactive           Status = "INACTIVE"
)

type Job struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
	URL    string `json:"url" binding:"required"`
}
