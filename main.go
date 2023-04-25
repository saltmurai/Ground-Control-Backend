package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	missionv1 "github.com/saltmurai/drone-api-service/gen/mission/v1"
	"github.com/saltmurai/drone-api-service/gen/mission/v1/missionv1connect"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var port string

func init() {
	pflag.StringVarP(&port, "port", "p", ":3100", "start server on")
	pflag.Parse()
}

func main() {
	// init system logger
	log, _ := zap.NewProduction()
	defer log.Sync()
	sugar := log.Sugar()
	sugar.Infof("Starting server")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Use(middleware.RealIP)
	r.Use(middleware.Heartbeat("/ping"))

	path, handler := missionv1connect.NewMissionServiceHandler(&missionServiceServer{})
	r.Handle(path, handler)
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("oh no")
	})

	r.Route("/api", func(r chi.Router) {
		r.Get("/run", func(w http.ResponseWriter, r *http.Request) {

			w.Write([]byte("Run mission"))
		})
	})

	err := http.ListenAndServe(port, h2c.NewHandler(r, &http2.Server{}))
	if err != nil {
		sugar.Error(err)
	}
}

type missionServiceServer struct {
	missionv1connect.UnimplementedMissionServiceHandler
}

func (s *missionServiceServer) SendMission(
	ctx context.Context,
	req *connect.Request[missionv1.SendMissionRequest],
) (*connect.Response[missionv1.SendMissionResult], error) {
	id := req.Msg.GetId()
	init := req.Msg.GetInitInstructions()
	travel := req.Msg.GetTravelInstructions()
	action := req.Msg.GetActionInstructions()
	fmt.Printf("Got %s %v %v %v\n", id, init, travel, action)
	return connect.NewResponse(&missionv1.SendMissionResult{}), nil
}
