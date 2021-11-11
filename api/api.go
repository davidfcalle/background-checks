/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/temporalio/background-checks/queries"
	"github.com/temporalio/background-checks/types"
	"github.com/temporalio/background-checks/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

const DefaultEndpoint = "localhost:8081"
const TaskQueue = "background-checks-main"

func BackgroundCheckWorkflowID(email string) string {
	return fmt.Sprintf("BackgroundCheck-%s", email)
}

func CandidateWorkflowID(email string) string {
	return fmt.Sprintf("Candidate-%s", email)
}

func ResearcherWorkflowID(email string) string {
	return fmt.Sprintf("Researcher-%s", email)
}

func executeWorkflow(options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
	c, err := client.NewClient(client.Options{})
	if err != nil {
		return nil, err
	}
	defer c.Close()

	options.TaskQueue = TaskQueue

	return c.ExecuteWorkflow(
		context.Background(),
		options,
		workflows.BackgroundCheck,
		args...,
	)
}

func cancelWorkflow(wid string) error {
	c, err := client.NewClient(client.Options{})
	if err != nil {
		return err
	}
	defer c.Close()

	return c.CancelWorkflow(context.Background(), wid, "")
}

func completeActivity(token []byte, result interface{}, activityErr error) error {
	c, err := client.NewClient(client.Options{})
	if err != nil {
		return err
	}
	defer c.Close()

	return c.CompleteActivity(context.Background(), token, result, activityErr)
}

func queryWorkflow(wid string, queryType string, args ...interface{}) (converter.EncodedValue, error) {
	c, err := client.NewClient(client.Options{})
	if err != nil {
		return nil, err
	}
	defer c.Close()

	return c.QueryWorkflow(
		context.Background(),
		wid,
		"",
		queryType,
		args...,
	)
}

func handleCheckList(w http.ResponseWriter, r *http.Request) {
	checks := []types.BackgroundCheckInput{}

	// client.ListOpenWorkflowExecutions?

	json.NewEncoder(w).Encode(checks)
}

func handleCheckCreate(w http.ResponseWriter, r *http.Request) {
	var input types.BackgroundCheckInput

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = executeWorkflow(
		client.StartWorkflowOptions{
			ID: BackgroundCheckWorkflowID(input.Email),
		},
		workflows.BackgroundCheck,
		input,
	)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func handleCheckStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	email := vars["email"]

	v, err := queryWorkflow(
		BackgroundCheckWorkflowID(email),
		queries.BackgroundCheckStatus,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result types.BackgroundCheckStatus
	err = v.Get(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleCheckReport(w http.ResponseWriter, r *http.Request) {
}

func handleCheckConsent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	token, err := base64.StdEncoding.DecodeString(vars["token"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result types.ConsentResult
	err = json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = completeActivity(token, result, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleCheckDecline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	token, err := base64.StdEncoding.DecodeString(vars["token"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = completeActivity(token, types.ConsentResult{Consent: false}, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleCheckCancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]

	err := cancelWorkflow(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleCandidateTodoList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	email := vars["email"]

	v, err := queryWorkflow(
		CandidateWorkflowID(email),
		queries.CandidateTodosList,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result []types.CandidateTodo
	err = v.Get(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleResearcherTodoList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	email := vars["email"]

	v, err := queryWorkflow(
		ResearcherWorkflowID(email),
		queries.ResearcherTodosList,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result []types.ResearcherTodo
	err = v.Get(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleSaveSearchResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	token, err := base64.StdEncoding.DecodeString(vars["token"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result types.SearchResult
	err = json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = completeActivity(token, result.Result(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func Router() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/checks", handleCheckList).Name("checks_list")
	r.HandleFunc("/checks", handleCheckCreate).Methods("POST").Name("checks_create")
	r.HandleFunc("/checks/{email}", handleCheckStatus).Name("check")
	r.HandleFunc("/checks/{email}/cancel", handleCheckCancel).Methods("POST").Name("check_cancel")
	r.HandleFunc("/checks/{email}/report", handleCheckReport).Name("check_report")
	r.HandleFunc("/checks/{token}/consent", handleCheckConsent).Methods("POST").Name("check_consent")
	r.HandleFunc("/checks/{token}/decline", handleCheckDecline).Methods("POST").Name("check_decline")
	r.HandleFunc("/checks/{token}/search", handleSaveSearchResult).Methods("POST").Name("research_save")
	r.HandleFunc("/todos/candidate/{email}", handleCandidateTodoList).Name("todos_candidate")
	r.HandleFunc("/todos/researcher/{email}", handleResearcherTodoList).Name("todos_researcher")

	return r
}

func Run() {
	srv := &http.Server{
		Handler: Router(),
		Addr:    DefaultEndpoint,
	}

	log.Fatal(srv.ListenAndServe())
}
