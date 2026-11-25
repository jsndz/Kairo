package record

import "time"

// ProjectSummary provides lightweight project metadata for catalog listing and dashboards.
type ProjectSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Revision  uint64    `json:"revision"`
	Clips     int       `json:"clips"`
	Assets    int       `json:"assets"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
