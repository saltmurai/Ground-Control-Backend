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
	"github.com/rs/cors"
	pbjs "google.golang.org/protobuf/encoding/protojson"

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
	l  *zap.Logger
}

func (s *MissionServer) SendMission(
	ctx context.Context,
	req *connect.Request[missionv1.SendMissionRequest],
) (*connect.Response[missionv1.SendMissionResponse], error) {
	mes := req.Msg
	data, err := pbjs.Marshal(mes)
	if err != nil {
		s.l.Sugar().Error(err)
	}
	fmt.Println(string(data))
	jsonString := &missionv1.SendMissionRequest{}
	json.Unmarshal(data, jsonString)
	fmt.Print(jsonString)

	id := req.Msg.GetId()
	seq := req.Msg.SequenceItems

	// err = DialComm(&buf, s.l)
	// if err != nil {
	// 	s.l.Sugar().Error(err)
	// }

	return connect.NewResponse(&missionv1.SendMissionResponse{
		Success: true,
		Message: fmt.Sprintf("Send mission id %s with %d sequences", id, len(seq)),
	}), nil
}

func main() {
	// ctx := context.Background()

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
		return
	}
	defer db.Close()

	queries := gendb.New(db)
	missioner := &MissionServer{
		l:  log,
		db: queries,
	}

	mux := http.NewServeMux()
	r := chi.NewRouter()
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("testing"))
	})
	path, handler := missionv1connect.NewMissionServiceHandler(missioner)
	mux.Handle(path, handler)
	mux.Handle("/", r)

	server := cors.AllowAll().Handler(mux)

	err = http.ListenAndServe(":3002", h2c.NewHandler(server, &http2.Server{}))
	if err != nil {
		sugar.Error(err)
	}
}

func DialComm(buf *[]byte, l *zap.Logger) error {
	commConn, err := net.Dial("tcp4", "localhost:3003")
	if err != nil {
		l.Sugar().Error(err)
		return err
	}
	defer commConn.Close()

	_, err = commConn.Write(*buf)
	if err != nil {
		l.Sugar().Error(err)
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
			l.Sugar().Error(err)
			return err
		}
		resp = append(resp, buffer[:n]...)
	}

	l.Sugar().Info(string(resp))

	return nil
}
