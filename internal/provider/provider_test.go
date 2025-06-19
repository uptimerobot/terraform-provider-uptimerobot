// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"uptimerobot": providerserver.NewProtocol6WithError(New("test")()),
}

// mock server storage
var mockMonitors = make(map[int64]map[string]interface{})
var mockIntegrations = make(map[int64]map[string]interface{})
var mockMaintenanceWindows = make(map[int64]map[string]interface{})
var mockPSPs = make(map[int64]map[string]interface{})

// mockServer holds the test server and next ID counter
var mockServer *httptest.Server
var nextID int64 = 1

// setupMockServer creates a mock HTTP server for testing
func setupMockServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock monitor endpoints without /v3 prefix since the client base URL will be set without it for testing
	mux.HandleFunc("/monitors", handleMonitors)
	mux.HandleFunc("/monitors/", handleMonitorByID)

	// Mock integration endpoints
	mux.HandleFunc("/integrations", handleIntegrations)
	mux.HandleFunc("/integrations/", handleIntegrationByID)

	// Mock maintenance window endpoints
	mux.HandleFunc("/maintenance-windows", handleMaintenanceWindows)
	mux.HandleFunc("/maintenance-windows/", handleMaintenanceWindowByID)

	// Mock PSP endpoints
	mux.HandleFunc("/psps", handlePSPs)
	mux.HandleFunc("/psps/", handlePSPByID)

	return httptest.NewServer(mux)
}

// Mock handlers
func handleMonitors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		response := map[string]interface{}{
			"id":                       nextID,
			"friendlyName":             req["friendlyName"],
			"url":                      req["url"],
			"type":                     req["type"],
			"interval":                 req["interval"],
			"status":                   "active",
			"authType":                 req["authType"],
			"keywordCaseType":          req["keywordCaseType"],
			"checkSSLErrors":           req["checkSSLErrors"],
			"successHttpResponseCodes": req["successHttpResponseCodes"],
			"gracePeriod":              req["gracePeriod"],
			"sslExpirationReminder":    req["sslExpirationReminder"],
			"domainExpirationReminder": req["domainExpirationReminder"],
			"followRedirections":       req["followRedirections"],
			"customHttpHeaders":        map[string]string{},
			"assignedAlertContacts":    []map[string]interface{}{},
			"tags":                     []map[string]interface{}{},
		}

		// Store the monitor
		mockMonitors[nextID] = response
		nextID++

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case "GET":
		response := []map[string]interface{}{}
		for _, monitor := range mockMonitors {
			response = append(response, monitor)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleMonitorByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	idStr := r.URL.Path[len("/monitors/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		if monitor, exists := mockMonitors[id]; exists {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(monitor)
		} else {
			http.Error(w, "Monitor not found", http.StatusNotFound)
		}

	case "PATCH":
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if monitor, exists := mockMonitors[id]; exists {
			// Update the stored monitor
			for key, value := range req {
				monitor[key] = value
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(monitor)
		} else {
			http.Error(w, "Monitor not found", http.StatusNotFound)
		}

	case "DELETE":
		delete(mockMonitors, id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleIntegrations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		response := map[string]interface{}{
			"id":                     nextID,
			"friendlyName":           req["friendlyName"],
			"type":                   req["type"],
			"value":                  req["value"],
			"customValue":            req["customValue"],
			"enableNotificationsFor": req["enableNotificationsFor"],
			"sslExpirationReminder":  req["sslExpirationReminder"],
		}

		// Only include webhook-specific fields for webhook integrations
		if req["type"] == "webhook" {
			if sendAsJson, exists := req["sendAsJson"]; exists {
				response["sendAsJson"] = sendAsJson
			} else {
				response["sendAsJson"] = false
			}

			if sendAsQueryString, exists := req["sendAsQueryString"]; exists {
				response["sendAsQueryString"] = sendAsQueryString
			} else {
				response["sendAsQueryString"] = false
			}

			if postValue, exists := req["postValue"]; exists {
				response["postValue"] = postValue
			} else {
				response["postValue"] = ""
			}
		}

		// Store the integration
		mockIntegrations[nextID] = response
		nextID++

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleIntegrationByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	idStr := r.URL.Path[len("/integrations/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		if integration, exists := mockIntegrations[id]; exists {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(integration)
		} else {
			http.Error(w, "Integration not found", http.StatusNotFound)
		}

	case "PATCH":
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if integration, exists := mockIntegrations[id]; exists {
			// Update the stored integration
			for key, value := range req {
				integration[key] = value
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(integration)
		} else {
			http.Error(w, "Integration not found", http.StatusNotFound)
		}

	case "DELETE":
		delete(mockIntegrations, id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleMaintenanceWindows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		response := map[string]interface{}{
			"id":              nextID,
			"name":            req["name"],
			"interval":        req["interval"],
			"time":            req["time"],
			"duration":        req["duration"],
			"autoAddMonitors": req["autoAddMonitors"],
			"status":          "active",
		}

		// Store the maintenance window
		mockMaintenanceWindows[nextID] = response
		nextID++

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleMaintenanceWindowByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	idStr := r.URL.Path[len("/maintenance-windows/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		if mw, exists := mockMaintenanceWindows[id]; exists {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mw)
		} else {
			http.Error(w, "Maintenance window not found", http.StatusNotFound)
		}

	case "PATCH":
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if mw, exists := mockMaintenanceWindows[id]; exists {
			// Update the stored maintenance window
			for key, value := range req {
				mw[key] = value
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mw)
		} else {
			http.Error(w, "Maintenance window not found", http.StatusNotFound)
		}

	case "DELETE":
		delete(mockMaintenanceWindows, id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func handlePSPs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		response := map[string]interface{}{
			"id":                         nextID,
			"friendlyName":               req["friendlyName"],
			"monitorIds":                 req["monitorIds"],
			"status":                     "active",
			"urlKey":                     fmt.Sprintf("psp-%d", nextID),
			"isPasswordSet":              false,
			"shareAnalyticsConsent":      req["shareAnalyticsConsent"],
			"useSmallCookieConsentModal": req["useSmallCookieConsentModal"],
			"noIndex":                    req["noIndex"],
			"hideUrlLinks":               req["hideUrlLinks"],
			"showCookieBar":              req["showCookieBar"],
			"subscription":               false,
		}

		// Store the PSP
		mockPSPs[nextID] = response
		nextID++

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handlePSPByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	idStr := r.URL.Path[len("/psps/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		if psp, exists := mockPSPs[id]; exists {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(psp)
		} else {
			http.Error(w, "PSP not found", http.StatusNotFound)
		}

	case "PATCH":
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if psp, exists := mockPSPs[id]; exists {
			// Update the stored PSP
			for key, value := range req {
				psp[key] = value
			}

			// Ensure subscription field is always present
			if _, hasSubscription := psp["subscription"]; !hasSubscription {
				psp["subscription"] = false
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(psp)
		} else {
			http.Error(w, "PSP not found", http.StatusNotFound)
		}

	case "DELETE":
		delete(mockPSPs, id)
		w.WriteHeader(http.StatusNoContent)
	}
}

// Provider-level tests.
func TestProvider(t *testing.T) {
	p := New("test")()
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)
	if resp.TypeName != "uptimerobot" {
		t.Fatal("unexpected provider type name")
	}
}

func testAccPreCheck(t *testing.T) {
	// Setup mock server if not already running
	if mockServer == nil {
		mockServer = setupMockServer()
	}

	// Reset mock storage for clean state between tests
	mockMonitors = make(map[int64]map[string]interface{})
	mockIntegrations = make(map[int64]map[string]interface{})
	mockMaintenanceWindows = make(map[int64]map[string]interface{})
	mockPSPs = make(map[int64]map[string]interface{})
	nextID = 1

	// Set mock server URL as endpoint for testing
	os.Setenv("UPTIMEROBOT_ENDPOINT", mockServer.URL)

	// Ensure required environment variables are set for testing
	// These would normally be real credentials in CI/CD environments
	if os.Getenv("UPTIMEROBOT_API_KEY") == "" {
		os.Setenv("UPTIMEROBOT_API_KEY", "test-api-key")
	}
	if os.Getenv("UPTIMEROBOT_ORGANIZATION_ID") == "" {
		os.Setenv("UPTIMEROBOT_ORGANIZATION_ID", "1")
	}
}

// Cleanup function to stop mock server
func TestMain(m *testing.M) {
	// Setup mock server
	mockServer = setupMockServer()
	os.Setenv("UPTIMEROBOT_ENDPOINT", mockServer.URL)

	// Run tests
	code := m.Run()

	// Cleanup
	mockServer.Close()

	// Reset state between test runs
	mockMonitors = make(map[int64]map[string]interface{})
	mockIntegrations = make(map[int64]map[string]interface{})
	mockMaintenanceWindows = make(map[int64]map[string]interface{})
	mockPSPs = make(map[int64]map[string]interface{})
	nextID = 1

	os.Exit(code)
}

// Centralized provider configuration helper
func testAccProviderConfig() string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    uptimerobot = {
      source = "hashicorp/uptimerobot"
    }
  }
}

provider "uptimerobot" {
  api_key  = "test-api-key"
  endpoint = "%s"
}
`, os.Getenv("UPTIMEROBOT_ENDPOINT"))
}

// CheckDestroy functions for each resource type
func testAccCheckMonitorDestroy(s *terraform.State) error {
	// In a real implementation, we would check the actual API
	// For our mock implementation, we can check the mock storage
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_monitor" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			continue
		}

		// Check if the monitor still exists in our mock storage
		if _, exists := mockMonitors[id]; exists {
			return fmt.Errorf("Monitor (%s) still exists in mock storage", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckIntegrationDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_integration" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			continue
		}

		// Check if the integration still exists in our mock storage
		if _, exists := mockIntegrations[id]; exists {
			return fmt.Errorf("Integration (%s) still exists in mock storage", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckMaintenanceWindowDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_maintenance_window" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			continue
		}

		// Check if the maintenance window still exists in our mock storage
		if _, exists := mockMaintenanceWindows[id]; exists {
			return fmt.Errorf("Maintenance Window (%s) still exists in mock storage", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckPSPDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_psp" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			continue
		}

		// Check if the PSP still exists in our mock storage
		if _, exists := mockPSPs[id]; exists {
			return fmt.Errorf("PSP (%s) still exists in mock storage", rs.Primary.ID)
		}
	}
	return nil
}
