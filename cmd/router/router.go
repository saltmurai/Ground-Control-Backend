package router

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func CreateRouter(ctx context.Context) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/ws", handleWebSocket)

	// ================== DRONES API ==================
	r.Get("/drones", GetDrones)
	r.Post("/drones", AddDrone)
	r.Delete("/drones", DeleteDrone)
	r.Get("/activeDrones", Telemetry)
	r.Post("/resetDrones", ResetAllDrones)

	// ================== SEQUENCES API ==================
	r.Post("/sequences", AddSequences)
	r.Get("/sequences", GetSequences)

	// ================== MISSIONS API ==================
	r.Get("/missions", GetMissions)
	r.Post("/missions", AddMission)
	r.Delete("/missions", DeleteMission)
	r.Get("/mission/images/{id}", GetMissionImagePath)
	r.Post("/images", ServeImage)

	// ================== COMM API ==================
	r.Post("/sendMission/{id}", sendMissionWithID)
	r.Post("/confirmation/{id}/{flag}", sendFlagToDrone)
	r.Post("/upload/{id}", UploadImage)

	// ================== USERS API ==================
	r.Get("/users", GetUsers)
	r.Post("/users", AddUser)

	// ================== PACKAGES API ==================
	r.Get("/packages", GetPackages)
	r.Post("/packages", AddPackages)

	return r
}

func NotImpelmented(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Not Implemented"))
}
