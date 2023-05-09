package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/bufbuild/connect-go"
	"github.com/go-chi/chi/v5"

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
}

func (s *MissionServer) SendMission(
	ctx context.Context,
	req *connect.Request[missionv1.SendMissionRequest],
) (*connect.Response[missionv1.SendMissionResult], error) {
	id := req.Msg.GetId()
	seq := req.Msg.SequenceItems
	for _, item := range seq {
		fmt.Println(item.GetSequence())
	}

	return connect.NewResponse(&missionv1.SendMissionResult{
		Success: true,
		Message: fmt.Sprintf("Send mission id %s with %d sequences", id, len(seq)),
	}), nil
}

func main() {
	ctx := context.Background()

	log, _ := zap.NewProduction()
	defer log.Sync()
	sugar := log.Sugar()
	sugar.Infof("Starting server on 3002")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), 5432, os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_USER"))

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		sugar.Errorf("Can't connect to postgres: %s", err)
	}
	defer db.Close()

	queries := gendb.New(db)
	mission_name := "foo bar"
	nullMissionName := sql.NullString{
		String: mission_name,
		Valid:  true,
	}
	insertMission, err := queries.CreateMission(ctx, nullMissionName)
	if err != nil {
		sugar.Error(err)
	}
	sugar.Info(insertMission)

	listMissions, err := queries.ListMission(ctx)
	if err != nil {
		sugar.Error(err)
	}
	sugar.Info(listMissions)
	missioner := &MissionServer{}
	mux := http.NewServeMux()
	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("testing"))
	})
	path, handler := missionv1connect.NewMissionServiceHandler(missioner)
	mux.Handle(path, handler)
	mux.Handle("/", r)

	err = http.ListenAndServe(":3002", h2c.NewHandler(mux, &http2.Server{}))
	if err != nil {
		sugar.Error(err)
	}
}
