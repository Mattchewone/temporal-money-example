package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bojanz/httpx"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"

	"money-transfer-project-template-go/app"
)

var (
	HTTPPort = os.Getenv("PORT")
	temporal client.Client
)

// Ref: https://github.com/temporalio/temporal-ecommerce/blob/04aaba65a801e310b37469397eb6f8a97849dd60/workflow.go

func main() {
	var err error
	// Create the client object just once per process
	temporal, err = client.NewClient(client.Options{})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}

	r := mux.NewRouter()
	r.Handle("/account", http.HandlerFunc(CreateAccountHandler)).Methods("POST")
	r.Handle("/account/{workflowID}", http.HandlerFunc(GetAccountHandler)).Methods("GET")
	r.Handle("/account/{workflowID}/deposit", http.HandlerFunc(DepositToAccountHandler)).Methods("POST")
	r.Handle("/account/{workflowID}/withdraw", http.HandlerFunc(WithdrawToAccountHandler)).Methods("POST")

	r.NotFoundHandler = http.HandlerFunc(NotFoundHandler)

	var cors = handlers.CORS(handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}), handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}), handlers.AllowedOrigins([]string{"*"}))

	http.Handle("/", cors(r))
	server := httpx.NewServer(":"+HTTPPort, http.DefaultServeMux)
	server.WriteTimeout = time.Second * 240

	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	response, err := temporal.QueryWorkflowWithOptions(context.Background(), &client.QueryWorkflowWithOptionsRequest{
		WorkflowID: vars["workflowID"],
		RunID:      "",
		QueryType:  "getAccount",
		// QueryRejectCondition is an optional field used to reject queries based on workflow state.
		// QUERY_REJECT_CONDITION_NONE indicates that query should not be rejected.
		// QUERY_REJECT_CONDITION_NOT_OPEN indicates that query should be rejected if workflow is not open.
		// QUERY_REJECT_CONDITION_NOT_COMPLETED_CLEANLY indicates that query should be rejected if workflow did not complete cleanly (e.g. terminated, canceled timeout etc...).
		QueryRejectCondition: enumspb.QUERY_REJECT_CONDITION_NOT_OPEN,
	})
	if err != nil {
		fmt.Printf("Error here..")
		WriteError(w, err)
		return
	}
	var res interface{}
	if err := response.QueryResult.Get(&res); err != nil {
		WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	options := client.StartWorkflowOptions{
		ID:        uuid.New().String(),
		TaskQueue: app.TransferMoneyTaskQueue,
	}

	ac := app.Account{Amount: 0, Transfers: make([]app.TransferDetails, 0)}
	we, err := temporal.ExecuteWorkflow(context.Background(), options, app.TransferMoney, ac)
	if err != nil {
		WriteError(w, err)
		return
	}

	res := make(map[string]interface{})
	res["account"] = ac
	res["accountID"] = we.GetID()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func DepositToAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var transferDetails app.TransferDetails
	err := json.NewDecoder(r.Body).Decode(&transferDetails)
	if err != nil {
		WriteError(w, err)
		return
	}

	update := app.MoneySignal{Route: app.RouteTypes.DEPOSIT, TransferDetails: transferDetails}

	err = temporal.SignalWorkflow(context.Background(), vars["workflowID"], "", app.SignalChannelName, update)
	if err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	res := make(map[string]interface{})
	res["ok"] = 1
	json.NewEncoder(w).Encode(res)
}

func WithdrawToAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var transferDetails app.TransferDetails
	err := json.NewDecoder(r.Body).Decode(&transferDetails)
	if err != nil {
		WriteError(w, err)
		return
	}

	update := app.MoneySignal{Route: app.RouteTypes.WITHDRAW, TransferDetails: transferDetails}

	err = temporal.SignalWorkflow(context.Background(), vars["workflowID"], "", app.SignalChannelName, update)
	if err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	res := make(map[string]interface{})
	res["ok"] = 1
	json.NewEncoder(w).Encode(res)
}

type ErrorResponse struct {
	Message string
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	res := ErrorResponse{Message: "Endpoint not found"}
	json.NewEncoder(w).Encode(res)
}

func WriteError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	res := ErrorResponse{Message: err.Error()}
	json.NewEncoder(w).Encode(res)
}
