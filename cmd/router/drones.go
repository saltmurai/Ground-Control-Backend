package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/saltmurai/drone-api-service/cmd/database"
	"github.com/saltmurai/drone-api-service/gendb"
	"go.uber.org/zap"
)

type DroneWithTelemetries struct {
	gendb.Drone
	Position string `json:"Position"`
	Battery  string `json:"battery"`
}

func AddDrone(w http.ResponseWriter, r *http.Request) {
	// parse body
	queries := database.GetQueries()
	drones := gendb.Drone{}
	err := json.NewDecoder(r.Body).Decode(&drones)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// insert to db
	_, err = queries.InsertDrone(r.Context(), gendb.InsertDroneParams{
		Name:    drones.Name,
		Address: drones.Address,
		Ip:      drones.Ip,
		Status:  false,
	})
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func DeleteDrone(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	// get Id from body
	drones := gendb.Drone{}
	err := json.NewDecoder(r.Body).Decode(&drones)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// delete from db
	_, err = queries.DeleteDrone(r.Context(), drones.ID)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func GetDrones(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	// get all drones
	drones, err := queries.ListDrones(r.Context())
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// convert to json
	jsonString, err := json.Marshal(drones)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(jsonString)
}

func Telemetry(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	queries := database.GetQueries()
	redisClient := database.GetRedisClient()
	activeDrones, err := queries.ListActiveDrones(r.Context())
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	activeDronesWithTelemetries := make([]DroneWithTelemetries, 0)
	for _, drone := range activeDrones {
		positionKey := fmt.Sprintf("%d-%s", drone.ID, "postion")
		batteryKey := fmt.Sprintf("%d-%s", drone.ID, "battery")

		position, err := redisClient.Get(ctx, positionKey).Result()
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		battery, err := redisClient.Get(ctx, batteryKey).Result()
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		activeDronesWithTelemetries = append(activeDronesWithTelemetries, DroneWithTelemetries{
			Drone:    drone,
			Position: position,
			Battery:  battery,
		})
	}

	jsonString, err := json.Marshal(activeDronesWithTelemetries)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(jsonString)
}

func ResetAllDrones(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	res, err := queries.ResetAllDroneStatus(r.Context())
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonString, err := json.Marshal(res)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(jsonString)
}
