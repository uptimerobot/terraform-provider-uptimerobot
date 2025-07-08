package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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

// mock server storage.
var mockMonitors = make(map[int64]map[string]interface{})
var mockIntegrations = make(map[int64]map[string]interface{})
var mockMaintenanceWindows = make(map[int64]map[string]interface{})
var mockPSPs = make(map[int64]map[string]interface{})

// mockServer holds the test server and next ID counter.
var mockServer *httptest.Server
var nextID int64 = 1

// setupMockServer creates a mock HTTP server for testing.
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

// Mock handlers.

// Helper function to validate enum values.
func isValidEnumValue(value interface{}, validValues []string) bool {
	if str, ok := value.(string); ok {
		for _, valid := range validValues {
			if str == valid {
				return true
			}
		}
	}
	return false
}

func handleMonitors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate monitor type
		if monitorType, exists := req["type"]; exists {
			validTypes := []string{"HTTP", "KEYWORD", "PING", "PORT", "HEARTBEAT", "DNS"}
			if !isValidEnumValue(monitorType, validTypes) {
				http.Error(w, "Invalid monitor type", http.StatusBadRequest)
				return
			}
		}

		// Validate friendlyName length
		if friendlyName, exists := req["friendlyName"]; exists {
			if str, ok := friendlyName.(string); ok && len(str) > 100 {
				http.Error(w, "friendlyName must be 100 characters or less", http.StatusBadRequest)
				return
			}
		}

		response := map[string]interface{}{
			"id":                       nextID,
			"friendlyName":             req["friendlyName"],
			"url":                      req["url"],
			"type":                     req["type"],
			"interval":                 req["interval"],
			"status":                   "Active",
			"authType":                 req["authType"],
			"keywordCaseType":          req["keywordCaseType"],
			"checkSSLErrors":           req["checkSSLErrors"],
			"successHttpResponseCodes": req["successHttpResponseCodes"],
			"gracePeriod":              req["gracePeriod"],
			"sslExpirationReminder":    req["sslExpirationReminder"],
			"domainExpirationReminder": req["domainExpirationReminder"],
			"followRedirections":       req["followRedirections"],
			"customHttpHeaders":        map[string]string{},
			"assignedAlertContacts":    convertAlertContactsFromRequest(req["assignedAlertContacts"]),
			"tags":                     convertTagsFromRequest(req["tagNames"]),
			"maintenanceWindows":       convertMaintenanceWindowsFromRequest(req["maintenanceWindowsIds"]),
		}

		// Add optional fields if they exist in the request
		if keywordType, exists := req["keywordType"]; exists {
			response["keywordType"] = keywordType
		}
		if keywordValue, exists := req["keywordValue"]; exists {
			response["keywordValue"] = keywordValue
		}
		if responseTimeThreshold, exists := req["responseTimeThreshold"]; exists {
			response["responseTimeThreshold"] = responseTimeThreshold
		}
		if regionalData, exists := req["regionalData"]; exists {
			// Convert simple string to API format expected by provider
			if regionalDataStr, ok := regionalData.(string); ok {
				response["regionalData"] = map[string]interface{}{
					"REGION": []string{regionalDataStr},
				}
			}
		}
		if port, exists := req["port"]; exists {
			response["port"] = port
		}
		if httpMethodType, exists := req["httpMethodType"]; exists {
			response["httpMethodType"] = httpMethodType
		}
		if postValueData, exists := req["postValueData"]; exists {
			response["postValueData"] = postValueData
		}
		if postValueType, exists := req["postValueType"]; exists {
			response["postValueType"] = postValueType
		}
		if httpUsername, exists := req["httpUsername"]; exists {
			response["httpUsername"] = httpUsername
		}
		if httpPassword, exists := req["httpPassword"]; exists {
			response["httpPassword"] = httpPassword
		}
		if timeout, exists := req["timeout"]; exists {
			response["timeout"] = timeout
		}
		if regionalData, exists := req["regionalData"]; exists {
			response["regionalData"] = regionalData
		}

		// Store the monitor
		mockMonitors[nextID] = response
		nextID++

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

	case "GET":
		response := []map[string]interface{}{}
		for _, monitor := range mockMonitors {
			response = append(response, monitor)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func handleMonitorByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	idStr := r.URL.Path[len("/monitors/"):]

	// Check if this is a reset request
	if strings.HasSuffix(idStr, "/reset") {
		// Extract the actual monitor ID
		actualIDStr := strings.TrimSuffix(idStr, "/reset")
		id, err := strconv.ParseInt(actualIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if r.Method == "POST" {
			if monitor, exists := mockMonitors[id]; exists {
				// Reset monitor stats (simulate the reset operation)
				// In a real API, this would reset statistics, incidents, alerts etc.
				monitor["lastResetDate"] = time.Now().Unix()

				w.WriteHeader(http.StatusCreated) // API spec expects 201
				return
			} else {
				http.Error(w, "Monitor not found", http.StatusNotFound)
				return
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	// Normal monitor operations
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		if monitor, exists := mockMonitors[id]; exists {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(monitor); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Monitor not found", http.StatusNotFound)
		}

	case "PATCH":
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if monitor, exists := mockMonitors[id]; exists {
			// Special handling for assignedAlertContacts
			// If not present in request, assume user wants to remove them
			if _, hasAlertContacts := req["assignedAlertContacts"]; !hasAlertContacts {
				// Not present in request = remove alert contacts
				monitor["assignedAlertContacts"] = []map[string]interface{}{}
			}

			// Special handling for tags
			// If not present in request, assume user wants to remove them
			if _, hasTags := req["tagNames"]; !hasTags {
				// Not present in request = remove tags
				monitor["tags"] = []map[string]interface{}{}
			}

			// Special handling for maintenanceWindows
			// If not present in request, assume user wants to remove them
			if _, hasMaintenanceWindows := req["maintenanceWindowsIds"]; !hasMaintenanceWindows {
				// Not present in request = remove maintenance windows
				monitor["maintenanceWindows"] = []map[string]interface{}{}
			}

			// Handle optional fields - remove if not present in update
			if _, hasResponseTimeThreshold := req["responseTimeThreshold"]; !hasResponseTimeThreshold {
				delete(monitor, "responseTimeThreshold")
			}
			if _, hasRegionalData := req["regionalData"]; !hasRegionalData {
				delete(monitor, "regionalData")
			}

			// Update the stored monitor
			for key, value := range req {
				switch key {
				case "assignedAlertContacts":
					// Convert alert contacts to the expected format
					monitor[key] = convertAlertContactsFromRequest(value)
				case "tagNames":
					// Convert tags to the expected format
					monitor["tags"] = convertTagsFromRequest(value)
				case "maintenanceWindowsIds":
					// Convert maintenance windows to the expected format
					monitor["maintenanceWindows"] = convertMaintenanceWindowsFromRequest(value)
				case "regionalData":
					// Convert simple string to API format expected by provider
					if regionalDataStr, ok := value.(string); ok {
						monitor["regionalData"] = map[string]interface{}{
							"REGION": []string{regionalDataStr},
						}
					}
				default:
					monitor[key] = value
				}
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(monitor); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate integration type
		if integrationType, exists := req["type"]; exists {
			validTypes := []string{"Slack", "Telegram", "MS Teams", "Webhook", "Zapier", "PagerDuty", "Google Chat", "Discord", "Splunk", "Pushbullet", "Pushover", "Email", "SMS"}
			if !isValidEnumValue(integrationType, validTypes) {
				http.Error(w, "Invalid integration type", http.StatusBadRequest)
				return
			}
		}

		// Validate friendlyName length
		if friendlyName, exists := req["friendlyName"]; exists {
			if str, ok := friendlyName.(string); ok && len(str) > 60 {
				http.Error(w, "friendlyName must be 60 characters or less", http.StatusBadRequest)
				return
			}
		}

		// Validate enableNotificationsFor - accept both integer and string values
		if notifFor, exists := req["enableNotificationsFor"]; exists {
			// Accept integers (from Terraform) and convert for validation
			if intVal, ok := notifFor.(float64); ok {
				// Integer values: 1=all, 2=down only, 3=up only, 0=none
				if intVal != 0 && intVal != 1 && intVal != 2 && intVal != 3 {
					http.Error(w, "Invalid enableNotificationsFor value", http.StatusBadRequest)
					return
				}
			} else if strVal, ok := notifFor.(string); ok {
				validValues := []string{"UpAndDown", "Down", "Up", "None"}
				if !isValidEnumValue(strVal, validValues) {
					http.Error(w, "Invalid enableNotificationsFor value", http.StatusBadRequest)
					return
				}
			} else {
				http.Error(w, "Invalid enableNotificationsFor value", http.StatusBadRequest)
				return
			}
		}

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
		if req["type"] == "Webhook" {
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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
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
			if err := json.NewEncoder(w).Encode(integration); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Integration not found", http.StatusNotFound)
		}

	case "PATCH":
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if integration, exists := mockIntegrations[id]; exists {
			// Update the stored integration
			for key, value := range req {
				integration[key] = value
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(integration); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
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
	case "GET":
		// List maintenance windows with pagination support
		cursor := r.URL.Query().Get("cursor")
		var startID int64 = 0
		if cursor != "" {
			if parsedCursor, err := strconv.ParseInt(cursor, 10, 64); err == nil {
				startID = parsedCursor
			}
		}

		response := map[string]interface{}{
			"data": []map[string]interface{}{},
			"pagination": map[string]interface{}{
				"cursor":  nil,
				"hasMore": false,
			},
		}

		windows := []map[string]interface{}{}
		for id, window := range mockMaintenanceWindows {
			if id >= startID {
				windows = append(windows, window)
			}
		}

		response["data"] = windows

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

	case "POST":
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate name length
		if name, exists := req["name"]; exists {
			if str, ok := name.(string); ok && len(str) > 100 {
				http.Error(w, "name must be 100 characters or less", http.StatusBadRequest)
				return
			}
		}

		// Validate duration - should be a simple integer in minutes
		if durationReq, exists := req["duration"]; exists {
			if _, ok := durationReq.(float64); !ok {
				if _, ok := durationReq.(int); !ok {
					http.Error(w, "duration must be a number (minutes)", http.StatusBadRequest)
					return
				}
			}
		}

		response := map[string]interface{}{
			"id":              nextID,
			"name":            req["name"],
			"interval":        req["interval"],
			"time":            req["time"],
			"duration":        req["duration"], // Keep as simple integer
			"autoAddMonitors": req["autoAddMonitors"],
			"status":          "Active",
		}

		// Store the maintenance window
		mockMaintenanceWindows[nextID] = response
		nextID++

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
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
			if err := json.NewEncoder(w).Encode(mw); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Maintenance window not found", http.StatusNotFound)
		}

	case "PATCH":
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if mw, exists := mockMaintenanceWindows[id]; exists {
			// Update the stored maintenance window
			for key, value := range req {
				mw[key] = value
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mw); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate friendlyName length
		if friendlyName, exists := req["friendlyName"]; exists {
			if str, ok := friendlyName.(string); ok && len(str) > 100 {
				http.Error(w, "friendlyName must be 100 characters or less", http.StatusBadRequest)
				return
			}
		}

		response := map[string]interface{}{
			"id":                         nextID,
			"friendlyName":               req["friendlyName"],
			"monitorIds":                 req["monitorIds"],
			"status":                     "Active", // ✅ Fixed: Use proper case
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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
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
			if err := json.NewEncoder(w).Encode(psp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "PSP not found", http.StatusNotFound)
		}

	case "PATCH":
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

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
			if err := json.NewEncoder(w).Encode(psp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
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

func testAccPreCheck() {
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
	if err := os.Setenv("UPTIMEROBOT_ENDPOINT", mockServer.URL); err != nil {
		panic(fmt.Sprintf("Failed to set UPTIMEROBOT_ENDPOINT: %v", err))
	}

	// Ensure required environment variables are set for testing
	// These would normally be real credentials in CI/CD environments
	if os.Getenv("UPTIMEROBOT_API_KEY") == "" {
		if err := os.Setenv("UPTIMEROBOT_API_KEY", "test-api-key"); err != nil {
			panic(fmt.Sprintf("Failed to set UPTIMEROBOT_API_KEY: %v", err))
		}
	}
	if os.Getenv("UPTIMEROBOT_ORGANIZATION_ID") == "" {
		if err := os.Setenv("UPTIMEROBOT_ORGANIZATION_ID", "1"); err != nil {
			panic(fmt.Sprintf("Failed to set UPTIMEROBOT_ORGANIZATION_ID: %v", err))
		}
	}
}

// Cleanup function to stop mock server.
func TestMain(m *testing.M) {
	// Setup mock server
	mockServer = setupMockServer()
	if err := os.Setenv("UPTIMEROBOT_ENDPOINT", mockServer.URL); err != nil {
		panic(fmt.Sprintf("Failed to set UPTIMEROBOT_ENDPOINT: %v", err))
	}

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

// Centralized provider configuration helper.
// convertAlertContactsFromRequest converts the request format to the API response format.
func convertAlertContactsFromRequest(requestContacts interface{}) []map[string]interface{} {
	if requestContacts == nil {
		return []map[string]interface{}{}
	}

	// Handle the case where it's already a slice of contact IDs
	if contactIDs, ok := requestContacts.([]interface{}); ok {
		result := make([]map[string]interface{}, len(contactIDs))
		for i, id := range contactIDs {
			if idStr, ok := id.(string); ok {
				result[i] = map[string]interface{}{
					"alertContactId": idStr,
					"threshold":      1,
					"recurrence":     1,
				}
			}
		}
		return result
	}

	return []map[string]interface{}{}
}

// convertTagsFromRequest converts the request format to the API response format.
func convertTagsFromRequest(requestTags interface{}) []map[string]interface{} {
	if requestTags == nil {
		return []map[string]interface{}{}
	}

	// Handle the case where it's already a slice of tag names
	if tagNames, ok := requestTags.([]interface{}); ok {
		result := make([]map[string]interface{}, len(tagNames))
		for i, name := range tagNames {
			if nameStr, ok := name.(string); ok {
				result[i] = map[string]interface{}{
					"name": nameStr,
				}
			}
		}
		return result
	}

	return []map[string]interface{}{}
}

// convertMaintenanceWindowsFromRequest converts the request format to the API response format.
func convertMaintenanceWindowsFromRequest(requestWindows interface{}) []map[string]interface{} {
	if requestWindows == nil {
		return []map[string]interface{}{}
	}

	// Handle the case where it's already a slice of maintenance window IDs
	if windowIDs, ok := requestWindows.([]interface{}); ok {
		result := make([]map[string]interface{}, len(windowIDs))
		for i, id := range windowIDs {
			// Convert different numeric types to int64
			var idInt64 int64
			switch v := id.(type) {
			case int:
				idInt64 = int64(v)
			case int64:
				idInt64 = v
			case float64:
				idInt64 = int64(v)
			default:
				continue // Skip invalid types
			}

			result[i] = map[string]interface{}{
				"id":   idInt64,
				"name": fmt.Sprintf("Maintenance Window %d", idInt64),
			}
		}
		return result
	}

	return []map[string]interface{}{}
}

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
  api_url = "%s"
}
`, os.Getenv("UPTIMEROBOT_ENDPOINT"))
}

// CheckDestroy functions for each resource type.
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
