package handlers

import (
	"sync/atomic"

	"github.com/IdrisovMarat/httpserver/internal/database"
)

type ApiConfig struct {
	FileserverHits atomic.Int32
	Db             *database.Queries
	Platform       string
	JWTsecret      string
	PolkaKey       string
}
