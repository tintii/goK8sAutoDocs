package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPageRoutes(t *testing.T) {
	mux := http.NewServeMux()
	registerPageRoutes(mux)
	for route, file := range map[string]string{
		"/":                                "autodocs-dashboard.html",
		"/?view=discovered&cluster=remote": "autodocs-dashboard.html",
		"/autodocs-dashboard-page?view=discovered&environment=dev": "autodocs-dashboard.html",
		"/versions-dashboard-page":                                 "versions-dashboard.html",
		"/overview-page":                                           "index.html",
	} {
		t.Run(route, func(t *testing.T) {
			expected, err := os.ReadFile("static/" + file)
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest("GET", route, nil))
			if response.Code != http.StatusOK || response.Body.String() != string(expected) {
				t.Fatalf("%s: expected %s, got status %d", route, file, response.Code)
			}
		})
	}
	for _, route := range []string{"/unknown", "/autodocs-dashboard-page/extra", "/overview-page/extra", "/api/v1/unknown"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest("GET", route, nil))
		if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "<!DOCTYPE html>") {
			t.Fatalf("%s: expected 404, got %d", route, response.Code)
		}
	}
}
