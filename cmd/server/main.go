package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/bufbuild/connect-go"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/cors"
	pbjs "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	missionv1 "github.com/saltmurai/drone-api-service/gen/mission/v1"
	"github.com/saltmurai/drone-api-service/gen/mission/v1/missionv1connect"
	"github.com/saltmurai/drone-api-service/gendb"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// MissionServer is an interface for the missionn define in proto file
type MissionServer struct {
	missionv1connect.UnimplementedMissionServiceHandler
	db *gendb.Queries
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

	err = DialComm(&buf)
	if err != nil {
		zap.L().Sugar().Error(err)
	}

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
	opt, err := redis.ParseURL("redis://redis:6379")
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
		_, err = queries.InsertMission(r.Context(), gendb.InsertMissionParams{
			Name:        missions.Name,
			SeqID:       missions.SeqID,
			DroneID:     missions.DroneID,
			PackageID:   missions.PackageID,
			ImageFolder: fmt.Sprintf("%s-%d", missions.Name, missions.DroneID),
			Status:      false,
		})
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
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

func DialComm(buf *[]byte) error {
	commConn, err := net.Dial("tcp4", "localhost:3003")
	if err != nil {
		zap.L().Sugar().Errorf("Error dialing to comm service")
		return err
	}
	defer commConn.Close()

	_, err = commConn.Write(*buf)
	if err != nil {
		zap.L().Sugar().Error(err)
		return err
	}

	resp := make([]byte, 0)
	buffer := make([]byte, 1024)

	for {
		n, err := commConn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			zap.L().Sugar().Error(err)
			return err
		}
		resp = append(resp, buffer[:n]...)
	}

	zap.L().Sugar().Info(string(resp))

	return nil
}
