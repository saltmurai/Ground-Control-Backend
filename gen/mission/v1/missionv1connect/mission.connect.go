// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: mission/v1/mission.proto

package missionv1connect

import (
	context "context"
	errors "errors"
	connect_go "github.com/bufbuild/connect-go"
	v1 "github.com/saltmurai/drone-api-service/gen/mission/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// MissionServiceName is the fully-qualified name of the MissionService service.
	MissionServiceName = "mission.v1.MissionService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// MissionServiceSendMissionProcedure is the fully-qualified name of the MissionService's
	// SendMission RPC.
	MissionServiceSendMissionProcedure = "/mission.v1.MissionService/SendMission"
)

// MissionServiceClient is a client for the mission.v1.MissionService service.
type MissionServiceClient interface {
	// Send a mission to the drone.
	SendMission(context.Context, *connect_go.Request[v1.SendMissionRequest]) (*connect_go.Response[v1.SendMissionResult], error)
}

// NewMissionServiceClient constructs a client for the mission.v1.MissionService service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewMissionServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) MissionServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &missionServiceClient{
		sendMission: connect_go.NewClient[v1.SendMissionRequest, v1.SendMissionResult](
			httpClient,
			baseURL+MissionServiceSendMissionProcedure,
			opts...,
		),
	}
}

// missionServiceClient implements MissionServiceClient.
type missionServiceClient struct {
	sendMission *connect_go.Client[v1.SendMissionRequest, v1.SendMissionResult]
}

// SendMission calls mission.v1.MissionService.SendMission.
func (c *missionServiceClient) SendMission(ctx context.Context, req *connect_go.Request[v1.SendMissionRequest]) (*connect_go.Response[v1.SendMissionResult], error) {
	return c.sendMission.CallUnary(ctx, req)
}

// MissionServiceHandler is an implementation of the mission.v1.MissionService service.
type MissionServiceHandler interface {
	// Send a mission to the drone.
	SendMission(context.Context, *connect_go.Request[v1.SendMissionRequest]) (*connect_go.Response[v1.SendMissionResult], error)
}

// NewMissionServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewMissionServiceHandler(svc MissionServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle(MissionServiceSendMissionProcedure, connect_go.NewUnaryHandler(
		MissionServiceSendMissionProcedure,
		svc.SendMission,
		opts...,
	))
	return "/mission.v1.MissionService/", mux
}

// UnimplementedMissionServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedMissionServiceHandler struct{}

func (UnimplementedMissionServiceHandler) SendMission(context.Context, *connect_go.Request[v1.SendMissionRequest]) (*connect_go.Response[v1.SendMissionResult], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("mission.v1.MissionService.SendMission is not implemented"))
}