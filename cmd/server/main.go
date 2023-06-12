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

	id := req.Msg.GetId()
	seq := req.Msg.SequenceItems

	err = DialComm(&buf)
	if err != nil {
		zap.L().Sugar().Error(err)
	}

	return connect.NewResponse(&missionv1.SendMissionResponse{
		Success: true,
		Message: fmt.Sprintf("Send mission id %s with %d sequences", id, len(seq)),
	}), nil
}

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

func main() {
	// ctx := context.Background()

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

		drones.ID = uuid.New()

		// insert to db
		_, err = queries.InsertDrone(r.Context(), gendb.InsertDroneParams{
			ID:      drones.ID,
			Name:    drones.Name,
			Address: drones.Address,
			Status:  false,
		})
		if err != nil {
			zap.L().Sugar().Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
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
