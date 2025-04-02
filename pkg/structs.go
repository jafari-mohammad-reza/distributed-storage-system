package pkg

import "time"

type Storage struct {
	Id         string
	Index      int
	LastUpdate time.Time
	Port       int
}

type InvokeBody struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=16"`
	Agent    string `json:"agent"`
}
type InvokeResponse struct {
	Token string `json:"token"`
}

type ListUploadsResult struct {
	ID string
	FileName  string
	Directory string
	Versions  []UploadVersionResult
	CreatedAt string
}
type UploadVersionResult struct {
	ID        string
	CreatedAt string
}
