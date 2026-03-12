package types

import "time"

// TaskStatus represents task status
type TaskStatus string

const (
	StatusNEW        TaskStatus = "NEW"
	StatusRUNNING    TaskStatus = "RUNNING"
	StatusPROCESSING TaskStatus = "PROCESSING"
	StatusCANCELED   TaskStatus = "CANCELED"
	StatusDONE       TaskStatus = "DONE"
	StatusERROR      TaskStatus = "ERROR"
)

// IsValid checks if task status is valid
func (s TaskStatus) IsValid() bool {
	switch s {
	case StatusNEW, StatusRUNNING, StatusPROCESSING,
		StatusCANCELED, StatusDONE, StatusERROR:
		return true
	default:
		return false
	}
}

// IsTerminal checks if task status is terminal
func (s TaskStatus) IsTerminal() bool {
	return s == StatusDONE || s == StatusERROR || s == StatusCANCELED
}

// TaskStatusResponse represents task status API response
type TaskStatusResponse struct {
	Status int `json:"status"`
	Result struct {
		ID        string     `json:"id"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		Status    TaskStatus `json:"status"`
	} `json:"result"`
}
