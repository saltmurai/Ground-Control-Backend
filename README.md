# GROUND CONTROL STATION BACKEND AND COMMUNICATION SERVICE

This repo consist of backend for the Ground control station and communication module for the drone.

---

Presquites: Go >= 1.18. See the installation instruction ([Download and install - The Go Programming Language](https://go.dev/doc/install))

## How to run:

**Run the program**

```
go run main.go 
```

**Build and run the program**

```
go build main.go
./main
```

**run linting and formating**

```
make lint
```

---

## Status:

- Defined proto file for interchange communication between different program (like communication service, control service and front end). This file should be share between different codebase for this project.
  
- Implemented MissionServer that can be invoke by using gRPC protocol or traditional HTTP 1.1 work as well.
  
  ---
  
  ## Example sending a mission to the drone.
  
  Sequence_items is an array containing all init, action, travel.
  
  Using curl to send a mission sequence to the backend.
  
  ```json
  curl --request POST \
    --url http://localhost:3002/mission.v1.MissionService/SendMission \
    --header 'Content-Type: application/json' \
    --data '{
  	"id": "3",
  	"sequence_items": [
  		{
  			"init_sequence": {
  				"peripheral": [
  					1,
  					2,
  					3
  				],
  				"controller": "CONTROLLER_PX4_VELO_FB"
  			}
  		},
  		{
  			"travel_sequence": {
  				"planner": "PLANNER_MARKER",
  				"waypoint": [
  					2.13123123123123,
  					3.12415124145213,
  					3.2153124123135124
  				],
  				"constraint": [
  					2.1213123,
  					3.123123123,
  					4.312312312321331
  				],
  				"terminate": "TERMINATION_STD"
  			}
  		},
  		{
  			"action_sequence": {
  				"action": "ACTION_TAKEOFF",
  				"package": [
  					1,
  					2,
  					3,
  					4
  				],
  				"param": 2.41213123
  			}
  		},
  		{
  			"action_sequence": {
  				"action": "ACTION_TAKEOFF",
  				"package": [
  					1,
  					2,
  					3,
  					4
  				],
  				"param": 2.41213123
  			}
  		}
  	]
  }'
  ```
  
  Message return:
  
  ```json
  {
      "success":true,
      "message":"Send mission id 3 with 4 sequences"
  }
  ```