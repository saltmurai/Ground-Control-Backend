package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/bufbuild/connect-go"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/cors"
	pbjs "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	missionv1 "github.com/saltmurai/drone-api-service/gen/mission/v1"
	"github.com/saltmurai/drone-api-service/gen/mission/v1/missionv1connect"
	"github.com/saltmurai/drone-api-service/gendb"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// MissionServer is an interface for the missionn define in proto file
type MissionServer struct {
	missionv1connect.UnimplementedMissionServiceHandler
	db *gendb.Queries
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow any origin for WebSocket connections
	},
}

func (s *MissionServer) SendMission(
	ctx context.Context,
	req *connect.Request[missionv1.SendMissionRequest],
) (*connect.Response[missionv1.SendMissionResponse], error) {
	mes := req.Msg

	data, err := pbjs.Marshal(mes)
	if err != nil {
		zap.L().Sugar().Error(err)
	}

	buf, err := proto.Marshal(mes)
	if err != nil {
		zap.L().Sugar().Error(err)
	}

	fmt.Println(buf)
	fmt.Println(string(data))

	jsonString := &missionv1.SendMissionRequest{}
	json.Unmarshal(data, jsonString)
	fmt.Print(jsonString)

	seq := req.Msg.SequenceItems

	return connect.NewResponse(&missionv1.SendMissionResponse{
		Success: true,
		Message: fmt.Sprintf("Send mission id %d sequences", len(seq)),
	}), nil
}

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

type DroneWithTelemetries struct {
	gendb.Drone
	Position string `json:"Position"`
	Battery  string `json:"battery"`
}

func main() {
	ctx := context.Background()
	opt, err := redis.ParseURL(fmt.Sprintf("redis://%s:6379", os.Getenv("REDIS_HOST")))
	if err != nil {
		panic(err)
	}
	redisClient := redis.NewClient(opt)

	zap.L().Info("Starting server on 3002")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), 5432, os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_USER"))

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		zap.L().Sugar().Errorf("Can't connect DB")
		return
	}
	defer db.Close()

	queries := gendb.New(db)
	missioner := &MissionServer{
		db: queries,
	}

	mux := http.NewServeMux()
	// CRUD
	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("testing"))
	})
	r.Get("/ws", handleWebSocket)

	r.Post("/drones", func(w http.ResponseWriter, r *http.Request) {
		// parse body
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
	})

	r.Delete("/drones", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Get("/drones", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Get("/activeDrones", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Post("/resetDrones", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// ** Sequences **
	r.Post("/sequences", func(w http.ResponseWriter, r *http.Request) {
		// parse body
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
	})

	r.Get("/sequences", func(w http.ResponseWriter, r *http.Request) {
		// get all sequences
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
	})

	// ** Mission **
	r.Post("/missions", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Post("/sendMission/{id}", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Delete("/missions", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Post("/confirmation/{id}/{flag}", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Post("/upload/{id}", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Get("/missions", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Get("/mission/images/{id}", func(w http.ResponseWriter, r *http.Request) {
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

	})

	r.Post("/images", func(w http.ResponseWriter, r *http.Request) {
		type Image struct {
			Path string `json:"path"`
		}
		// get path from body
		image := Image{}
		err := json.NewDecoder(r.Body).Decode(&image)
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		http.ServeFile(w, r, image.Path)
	})

	r.Post("/users", func(w http.ResponseWriter, r *http.Request) {
		// parse body
		users := gendb.User{}
		err := json.NewDecoder(r.Body).Decode(&users)
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// generate uuid
		users.ID = uuid.New()
		// insert to db
		_, err = queries.InsertUser(r.Context(), gendb.InsertUserParams{
			ID:   users.ID,
			Name: users.Name,
		})
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		// get all users
		users, err := queries.ListUsers(r.Context())
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// convert to json
		jsonString, err := json.Marshal(users)
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jsonString)
	})

	r.Post("/packages", func(w http.ResponseWriter, r *http.Request) {
		// parse body
		packages := gendb.Package{}
		err := json.NewDecoder(r.Body).Decode(&packages)
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// add to db
		_, err = queries.InsertPackage(r.Context(), gendb.InsertPackageParams{
			Name:       packages.Name,
			Weight:     packages.Weight,
			Height:     packages.Height,
			Length:     packages.Length,
			SenderID:   packages.SenderID,
			ReceiverID: packages.ReceiverID,
		})
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	r.Get("/packages", func(w http.ResponseWriter, r *http.Request) {
		// get all packages
		packages, err := queries.ListPackages(r.Context())
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// convert to json
		jsonString, err := json.Marshal(packages)
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(jsonString)
	})

	// receive a image from drone then save to db

	path, handler := missionv1connect.NewMissionServiceHandler(missioner)
	mux.Handle(path, handler)
	mux.Handle("/", r)

	server := cors.AllowAll().Handler(mux)
	err = http.ListenAndServe(":3002", h2c.NewHandler(server, &http2.Server{}))

	if err != nil {
		zap.L().Sugar().Errorf("Can't start server")
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}

	// Connect to RabbitMQ
	connRabbitMQ, err := amqp.Dial(os.Getenv("AMQP_URL"))
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}
	defer connRabbitMQ.Close()

	// Create a channel and declare a queue
	channel, err := connRabbitMQ.Channel()
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}
	defer channel.Close()

	queue, err := channel.QueueDeclare(
		"log", // Queue name
		false, // Durable
		false, // Auto-deleted
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}

	// Consume messages from RabbitMQ
	messages, err := channel.Consume(
		queue.Name, // Queue name
		"",         // Consumer name
		true,       // Auto-acknowledge messages
		false,      // Exclusive
		false,      // No-local
		false,      // No-wait
		nil,        // Arguments
	)
	if err != nil {
		log.Println("RabbitMQ consume error:", err)
		return
	}

	// Forward RabbitMQ messages to WebSocket clients
	go func() {
		for message := range messages {
			err = conn.WriteMessage(websocket.TextMessage, message.Body)
			if err != nil {
				zap.L().Sugar().Error(err)
				break
			}
		}
	}()

	// Wait for WebSocket connection to close
	_, _, err = conn.ReadMessage()
	if err != nil {
		zap.L().Sugar().Error(err)
		//close websocket
		conn.Close()
		return
	}
}
