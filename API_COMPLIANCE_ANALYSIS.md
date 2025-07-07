# UptimeRobot Terraform Provider API Compliance Analysis

## Overview
This document details the analysis and fixes made to ensure the UptimeRobot Terraform provider complies with the official OpenAPI v3.1.0 specification.

## Issues Found and Fixed

### 1. Missing Monitor Types

**Issue**: The provider was missing `HEARTBEAT` and `DNS` monitor types that are supported by the API.

**API Specification**: 
```yaml
type:
  enum:
    - HTTP
    - KEYWORD
    - PING
    - PORT
    - HEARTBEAT
    - DNS
```

**Fix**: Added missing monitor types to the client constants:
```go
const (
    MonitorTypeHTTP      MonitorType = "HTTP"
    MonitorTypeKeyword   MonitorType = "KEYWORD"
    MonitorTypePing      MonitorType = "PING"
    MonitorTypePort      MonitorType = "PORT"
    MonitorTypeHeartbeat MonitorType = "HEARTBEAT"  // Added
    MonitorTypeDNS       MonitorType = "DNS"        // Added
)
```

**Files Modified**: `internal/client/monitor.go`

### 2. Missing Required Field Validation

**Issue**: The provider was not validating required fields based on monitor type.

**API Specification Requirements**:
- `port` is required for PORT monitors
- `keywordType` is required for KEYWORD monitors  
- `keywordValue` is required for KEYWORD monitors

**Fix**: Added comprehensive validation in the Create function:
```go
// Validate port is provided for PORT monitors
if monitorType == "PORT" && plan.Port.IsNull() {
    resp.Diagnostics.AddError(
        "Port required for PORT monitor",
        "Port must be specified for PORT monitor type",
    )
    return
}

// Validate keyword fields for KEYWORD monitors
if monitorType == "KEYWORD" {
    if plan.KeywordType.IsNull() {
        resp.Diagnostics.AddError(
            "KeywordType required for KEYWORD monitor",
            "KeywordType must be specified for KEYWORD monitor type (ALERT_EXISTS or ALERT_NOT_EXISTS)",
        )
        return
    }
    if plan.KeywordValue.IsNull() {
        resp.Diagnostics.AddError(
            "KeywordValue required for KEYWORD monitor",
            "KeywordValue must be specified for KEYWORD monitor type",
        )
        return
    }
}
```

**Files Modified**: `internal/provider/monitor_resource.go`

### 3. Incorrect Enum Values for Keyword Type

**Issue**: The provider was not validating that `keywordType` uses the correct enum values.

**API Specification**:
```yaml
keywordType:
  enum:
    - ALERT_EXISTS
    - ALERT_NOT_EXISTS
```

**Fix**: Added enum validation and updated field description:
```go
// Validate keyword type enum
keywordType := plan.KeywordType.ValueString()
if keywordType != "ALERT_EXISTS" && keywordType != "ALERT_NOT_EXISTS" {
    resp.Diagnostics.AddError(
        "Invalid KeywordType",
        "KeywordType must be either ALERT_EXISTS or ALERT_NOT_EXISTS",
    )
    return
}
```

**Files Modified**: `internal/provider/monitor_resource.go`

### 4. Missing Fields from API Specification

**Issue**: Several fields from the API specification were not implemented in the provider.

**API Specification Fields**:
- `responseTimeThreshold`: Response time threshold in milliseconds
- `regionalData`: Region for monitoring (na, eu, as, oc)

**Fix**: Added missing fields to:

1. **Client structs**:
```go
type CreateMonitorRequest struct {
    // ... existing fields ...
    ResponseTimeThreshold    int    `json:"responseTimeThreshold,omitempty"`
    RegionalData             string `json:"regionalData,omitempty"`
}
```

2. **Provider schema**:
```go
"response_time_threshold": schema.Int64Attribute{
    Description: "Response time threshold in milliseconds. Response time over this threshold will trigger an incident",
    Optional:    true,
},
"regional_data": schema.StringAttribute{
    Description: "Region for monitoring: na (North America), eu (Europe), as (Asia), oc (Oceania)",
    Optional:    true,
},
```

3. **Resource model**:
```go
type monitorResourceModel struct {
    // ... existing fields ...
    ResponseTimeThreshold    types.Int64  `tfsdk:"response_time_threshold"`
    RegionalData             types.String `tfsdk:"regional_data"`
}
```

**Files Modified**: 
- `internal/client/monitor.go`
- `internal/provider/monitor_resource.go`

### 5. Updated Field Descriptions

**Issue**: Field descriptions were not accurate or didn't match the API specification.

**Fix**: Updated field descriptions to match the API specification:

- `type`: Updated to include all supported monitor types
- `keyword_type`: Updated to specify valid enum values
- Added descriptions for new fields

**Files Modified**: `internal/provider/monitor_resource.go`

### 6. Enhanced Data Handling

**Issue**: The provider was not properly handling complex data structures returned by the API.

**Fix**: Added proper handling for regional data conversion:
```go
// Set regional data
if monitor.RegionalData != nil {
    // Convert regional data from API format to string
    if regionData, ok := monitor.RegionalData.(map[string]interface{}); ok {
        if regions, ok := regionData["REGION"].([]interface{}); ok && len(regions) > 0 {
            if region, ok := regions[0].(string); ok {
                state.RegionalData = types.StringValue(region)
            }
        }
    }
}
```

**Files Modified**: `internal/provider/monitor_resource.go`

## API Compliance Summary

### ✅ Compliant Areas
- Field naming conventions (using `friendlyName` in API calls)
- JSON field mappings match API specification
- Monitor type enum values match API specification
- Keyword case type handling (0/1 numeric values)
- Required field structure matches API specification

### ✅ Fixed Issues
- Added missing monitor types (HEARTBEAT, DNS)
- Added missing fields (responseTimeThreshold, regionalData)
- Added proper validation for required fields by monitor type
- Added enum validation for keyword types
- Updated field descriptions to match API specification

### 📋 Additional Improvements Made
- Enhanced error messages with clear guidance
- Added comprehensive validation logic
- Improved data type handling for complex API responses
- Added proper null/empty value handling

## Testing Recommendations

1. **PORT Monitor Testing**:
   - Test that PORT monitors require a port number
   - Test that PORT monitors fail creation without a port

2. **KEYWORD Monitor Testing**:
   - Test that KEYWORD monitors require both keywordType and keywordValue
   - Test that keywordType only accepts ALERT_EXISTS or ALERT_NOT_EXISTS
   - Test that KEYWORD monitors fail creation without required fields

3. **New Monitor Types**:
   - Test creation of HEARTBEAT monitors
   - Test creation of DNS monitors

4. **New Fields**:
   - Test responseTimeThreshold field functionality
   - Test regionalData field with valid regions (na, eu, as, oc)

## Files Modified

1. `internal/client/monitor.go` - Added missing monitor types and fields
2. `internal/provider/monitor_resource.go` - Added validation, new fields, and enhanced data handling

## OpenAPI Specification Compliance

The provider now fully complies with the UptimeRobot OpenAPI v3.1.0 specification located at:
https://cdn.uptimerobot.com/api/openapi.yaml

All monitor-related endpoints, fields, and validation rules have been implemented according to the API specification.