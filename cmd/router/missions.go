package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/saltmurai/drone-api-service/cmd/database"
	"github.com/saltmurai/drone-api-service/gendb"
	"go.uber.org/zap"
)

func GetMissions(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	// get all missions
	missions, err := queries.ListMissions(r.Context())
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// convert to json
	jsonString, err := json.Marshal(missions)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(jsonString)
}

func AddMission(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	// parse body
	missions := gendb.Mission{}
	err := json.NewDecoder(r.Body).Decode(&missions)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// insert to db
	created, err := queries.InsertMission(r.Context(), gendb.InsertMissionParams{
		Name:      missions.Name,
		SeqID:     missions.SeqID,
		DroneID:   missions.DroneID,
		PackageID: missions.PackageID,
		Status:    "pending",
	})
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = queries.UpdateMissionImageFolder(r.Context(), gendb.UpdateMissionImageFolderParams{
		ID:          created.ID,
		ImageFolder: fmt.Sprintf("%s-%d", created.Name, created.ID),
	})
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// created image folder
	folderPath := fmt.Sprintf("./images/%s-%d", created.Name, created.ID)
	err = os.MkdirAll(folderPath, 0755)
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Can't create image folder", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func sendMissionWithID(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
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

	seq, err := queries.GetSequenceByID(r.Context(), mission.SeqID)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	requestData := map[string]interface{}{
		"id":                    mission.ID,
		"name":                  mission.Name,
		"description":           seq.Description,
		"number_sequence_items": seq.Length,
		"sequences":             seq.Seq,
	}
	droneCommSerive := fmt.Sprintf("http://%s:5000/mission", mission.DroneIp)
	fmt.Println(droneCommSerive)
	// send the request to drone with request data
	json, err := json.Marshal(requestData)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp, err := http.Post(droneCommSerive, "application/json", bytes.NewBuffer(json))
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		zap.L().Sugar().Info("Can't send mission to drone, make sure it's turn on and comm serivce in running")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = queries.UpdateMissionStatus(r.Context(), gendb.UpdateMissionStatusParams{
		ID:     mission.ID,
		Status: "Sent to drone",
	})
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeleteMission(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	// get Id from body
	missions := gendb.Mission{}
	err := json.NewDecoder(r.Body).Decode(&missions)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// delete from db
	_, err = queries.DeleteMission(r.Context(), missions.ID)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func UploadImage(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
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

	// Get the image from the request body
	image, header, err := r.FormFile("image")
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Invalid image", http.StatusBadRequest)
		return
	}
	defer image.Close()

	// Create the folder if it doesn't exist
	folderPath := fmt.Sprintf("./images/%s", mission.ImageFolder)
	err = os.MkdirAll(folderPath, 0755)
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Can't create image folder", http.StatusInternalServerError)
		return
	}

	// Save the image to disk
	fileName := fmt.Sprintf("%s/%s", folderPath, header.Filename)
	out, err := os.Create(fileName)
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Can't save image", http.StatusBadRequest)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, image)
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Can't save image", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetMissionImagePath(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
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
	// return all images in the folder
	folderPath := fmt.Sprintf("./images/%s", mission.ImageFolder)
	files, err := os.ReadDir(folderPath)
	if err != nil {
		zap.L().Sugar().Error(err)
		http.Error(w, "Can't read image folder", http.StatusInternalServerError)
		return
	}
	// return image path
	imagePaths := make([]string, 0)
	for _, file := range files {
		imagePaths = append(imagePaths, fmt.Sprintf("%s/%s", folderPath, file.Name()))
	}
	jsonString, err := json.Marshal(imagePaths)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(jsonString)

}

type Image struct {
	Path string `json:"path"`
}

func ServeImage(w http.ResponseWriter, r *http.Request) {

	// get path from body
	image := Image{}
	err := json.NewDecoder(r.Body).Decode(&image)
	if err != nil {
		zap.L().Sugar().Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, image.Path)
}
