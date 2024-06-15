package main

import (
	"net/http"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

type myHandler struct{}

func (mh *myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

// go test -v
// go test -cover
// go test -coverprofile=coverage.out && go tool cover -html=coverage.out
// go tool cover -html=coverage.out
// go tool cover -func=coverage.out
