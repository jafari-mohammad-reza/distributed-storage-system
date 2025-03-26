package pkg

import "time"

type Storage struct {
	Id         string
	Index      int
	LastUpdate time.Time
}
