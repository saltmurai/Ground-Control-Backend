package router

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/saltmurai/drone-api-service/cmd/database"
	"github.com/saltmurai/drone-api-service/gendb"
	"go.uber.org/zap"
)

func AddUser(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
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
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
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
}
