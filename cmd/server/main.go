package main

import (
	"context"
	"net/http"

	"github.com/rs/cors"

	_ "github.com/lib/pq"
	"github.com/saltmurai/drone-api-service/cmd/database"
	"github.com/saltmurai/drone-api-service/cmd/router"
	"github.com/saltmurai/drone-api-service/gendb"
	"go.uber.org/zap"
)

// MissionServer is an interface for the missionn define in proto file
func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

type DroneWithTelemetries struct {
	gendb.Drone
	Position string `json:"Position"`
	Battery  string `json:"battery"`
}

func main() {
	var err error
	ctx := context.Background()
	zap.L().Info("Starting server on 3002")
	err = database.InitDatabase()
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}
	defer database.CloseDB()

	mux := http.NewServeMux()
	r := router.CreateRouter(ctx)
	mux.Handle("/", r)

	server := cors.AllowAll().Handler(mux)
	err = http.ListenAndServe(":3002", server)
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}
}
