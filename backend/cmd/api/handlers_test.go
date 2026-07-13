package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeployHandler(t *testing.T) {
	tempReq := DeployRequest{
		RepoURL: "github.com",
	}
	jsonBody, _ := json.Marshal(tempReq)
	req := httptest.NewRequest(http.MethodPost, "/deploy", bytes.NewBuffer(jsonBody))
	testQ := NewAsyncQueue()
	responseRecorder := httptest.NewRecorder()
	m := &MyAsyncQ{
		asyncQ: testQ,
	}
	m.DeployHandler(responseRecorder, req)
	if responseRecorder.Code != http.StatusAccepted {
		t.Errorf("expected status %d,got %d", http.StatusAccepted, responseRecorder.Code)
	}
	var res DeployResponse
	json.NewDecoder(responseRecorder.Body).Decode(&res)
	if res.Status != StateQueued {
		t.Errorf("expected status %v,got %v", StateQueued, res.Status)
	}
	if res.DeployID == "" {
		t.Errorf("expected a uuid but found %v", res.DeployID)
	}
}
