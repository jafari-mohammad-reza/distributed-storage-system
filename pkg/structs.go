package pkg

import "time"

type Storage struct {
	Id         string
	Index      int
	LastUpdate time.Time
}

type InvokeBody struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=16"`
	Agent    string `json:"agent"`
}
type InvokeResponse struct {
	Token string `json:"token"`
}
