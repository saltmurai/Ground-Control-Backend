package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/saltmurai/drone-api-service/cmd/database"
	"github.com/saltmurai/drone-api-service/gendb"
	"go.uber.org/zap"
)

// global queries variable

func AddSequences(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	sequences := gendb.Sequence{}
	err := json.NewDecoder(r.Body).Decode(&sequences)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// insert to db
	_, err = queries.InsertSequence(r.Context(), gendb.InsertSequenceParams{
		Name:        sequences.Name,
		Description: sequences.Description,
		Seq:         sequences.Seq,
		Length:      sequences.Length,
	})
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func GetSequences(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	sequences, err := queries.ListSequences(r.Context())
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// convert to json
	jsonString, err := json.Marshal(sequences)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(jsonString)
}

func sendFlagToDrone(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	flag := chi.URLParam(r, "flag")
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	mission, err := queries.GetMission(r.Context(), int64(id))
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Mission not found", http.StatusBadRequest)
		return
	}
	droneCommSerive := fmt.Sprintf("http://%s:5000/confirmation/%s", mission.DroneIp, flag)
	resp, err := http.Post(droneCommSerive, "application/json", nil)
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Drone not found", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		zap.L().Sugar().Info("Can't send confirmation to drone, make sure comm serivce and control service in running")
		http.Error(w, "Drone not found", http.StatusBadRequest)
		return
	}
	if flag == "FLAG_CONFIRM" {
		_, err = queries.UpdateMissionStatus(r.Context(), gendb.UpdateMissionStatusParams{
			ID:     mission.ID,
			Status: "MISSION CONFIRMED",
		})
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	if flag == "FLAG_REJECT" {
		_, err = queries.UpdateMissionStatus(r.Context(), gendb.UpdateMissionStatusParams{
			ID:     mission.ID,
			Status: "MISSION REJECTED",
		})
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
