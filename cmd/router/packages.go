package router

import (
	"encoding/json"
	"net/http"

	"github.com/saltmurai/drone-api-service/cmd/database"
	"github.com/saltmurai/drone-api-service/gendb"
	"go.uber.org/zap"
)

func AddPackages(w http.ResponseWriter, r *http.Request) {
	// parse body
	queries := database.GetQueries()
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
}

func GetPackages(w http.ResponseWriter, r *http.Request) {
	queries := database.GetQueries()
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
}
