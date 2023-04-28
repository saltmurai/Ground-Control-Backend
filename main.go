package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	missionv1 "github.com/saltmurai/drone-api-service/gen/mission/v1"
	"github.com/saltmurai/drone-api-service/gen/mission/v1/missionv1connect"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type timeHandler struct {
	format string
}

func (th timeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tm := time.Now().Format(th.format)
	w.Write([]byte("The time is: " + tm))
}

type MissionServer struct {
	missionv1connect.UnimplementedMissionServiceHandler
}

func (s *MissionServer) SendMission(
	ctx context.Context,
	req *connect.Request[missionv1.SendMissionRequest],
) (*connect.Response[missionv1.SendMissionResult], error) {
	id := req.Msg.GetId()
	init := req.Msg.GetInitInstructions()
	travel := req.Msg.GetTravelInstructions()
	fmt.Println(init.ProtoReflect())
	action := req.Msg.GetActionInstructions()
	fmt.Printf("Got %s %+v %v %v\n", id, init, travel, action)
	return connect.NewResponse(&missionv1.SendMissionResult{
		Success:      true,
		ErrorMessage: "none",
	}), nil
}

func main() {
	// init system logger
	log, _ := zap.NewProduction()
	defer log.Sync()
	sugar := log.Sugar()
	sugar.Infof("Starting server on 8080")

	missioner := &MissionServer{}
	mux := http.NewServeMux()
	path, handler := missionv1connect.NewMissionServiceHandler(missioner)
	mux.Handle(path, handler)

	th := timeHandler{format: time.RFC1123}

	mux.Handle("/time", th)

	err := http.ListenAndServe("localhost:3000", h2c.NewHandler(mux, &http2.Server{}))
	if err != nil {
		sugar.Error(err)
	}
}
