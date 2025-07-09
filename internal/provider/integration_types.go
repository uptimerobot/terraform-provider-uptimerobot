package provider

import "strings"

// IntegrationType represents the supported integration types.
type IntegrationType string

const (
	IntegrationTypeSlack      IntegrationType = "slack"
	IntegrationTypeWebhook    IntegrationType = "webhook"
	IntegrationTypeDiscord    IntegrationType = "discord"
	IntegrationTypeTelegram   IntegrationType = "telegram"
	IntegrationTypePushover   IntegrationType = "pushover"
	IntegrationTypePushbullet IntegrationType = "pushbullet"
	IntegrationTypeMSTeams    IntegrationType = "msteams"
	IntegrationTypeZapier     IntegrationType = "zapier"
	IntegrationTypePagerDuty  IntegrationType = "pagerduty"
	IntegrationTypeGoogleChat IntegrationType = "googlechat"
	IntegrationTypeSplunk     IntegrationType = "splunk"
	IntegrationTypeMattermost IntegrationType = "mattermost"
)

// AllIntegrationTypes returns a slice of all supported integration types.
func AllIntegrationTypes() []string {
	return []string{
		string(IntegrationTypeSlack),
		string(IntegrationTypeWebhook),
		string(IntegrationTypeDiscord),
		string(IntegrationTypeTelegram),
		string(IntegrationTypePushover),
		string(IntegrationTypePushbullet),
		string(IntegrationTypeMSTeams),
		string(IntegrationTypeZapier),
		string(IntegrationTypePagerDuty),
		string(IntegrationTypeGoogleChat),
		string(IntegrationTypeSplunk),
		string(IntegrationTypeMattermost),
	}
}

// IntegrationTypeDescriptions returns a map of integration types to their descriptions.
func IntegrationTypeDescriptions() map[string]string {
	return map[string]string{
		string(IntegrationTypeSlack):      "Slack webhook integration",
		string(IntegrationTypeWebhook):    "Custom webhook integration",
		string(IntegrationTypeDiscord):    "Discord webhook integration",
		string(IntegrationTypeTelegram):   "Telegram bot integration",
		string(IntegrationTypePushover):   "Pushover notifications",
		string(IntegrationTypePushbullet): "Pushbullet notifications",
		string(IntegrationTypeMSTeams):    "Microsoft Teams integration",
		string(IntegrationTypeZapier):     "Zapier integration",
		string(IntegrationTypePagerDuty):  "PagerDuty integration",
		string(IntegrationTypeGoogleChat): "Google Chat integration",
		string(IntegrationTypeSplunk):     "Splunk integration",
		string(IntegrationTypeMattermost): "Mattermost integration",
	}
}

// TransformIntegrationTypeToAPI converts case-insensitive terraform values to API format.
func TransformIntegrationTypeToAPI(terraformType string) string {
	switch strings.ToLower(terraformType) {
	case "slack":
		return "Slack"
	case "webhook":
		return "Webhook"
	case "discord":
		return "Discord"
	case "telegram":
		return "Telegram"
	case "pushover":
		return "Pushover"
	case "pushbullet":
		return "Pushbullet"
	case "msteams", "ms teams", "microsoft teams":
		return "MS Teams"
	case "zapier":
		return "Zapier"
	case "pagerduty", "pager duty":
		return "PagerDuty"
	case "googlechat", "google chat":
		return "Google Chat"
	case "splunk":
		return "Splunk"
	case "mattermost":
		return "Mattermost"
	default:
		return terraformType
	}
}

// TransformIntegrationTypeFromAPI converts API format to terraform format.
func TransformIntegrationTypeFromAPI(apiType string) string {
	switch apiType {
	case "Slack":
		return "slack"
	case "Webhook":
		return "webhook"
	case "Discord":
		return "discord"
	case "Telegram":
		return "telegram"
	case "Pushover":
		return "pushover"
	case "Pushbullet":
		return "pushbullet"
	case "MS Teams":
		return "msteams"
	case "Zapier":
		return "zapier"
	case "PagerDuty":
		return "pagerduty"
	case "Google Chat":
		return "googlechat"
	case "Splunk":
		return "splunk"
	case "Mattermost":
		return "mattermost"
	default:
		return strings.ToLower(apiType)
	}
}

// TransformIntegrationTypeToAPINumeric converts terraform format to API numeric type.
// Based on the API error message, these are the allowed values: 1, 2, 5, 6, 7, 8, 9, 11, 12, 13, 14, 15, 16, 17, 18, 20, 21, 23, 24.
func TransformIntegrationTypeToAPINumeric(terraformType string) int {
	switch strings.ToLower(terraformType) {
	case "webhook":
		return 2 // Common value for webhook integrations
	case "slack":
		return 1 // Common value for slack integrations
	case "discord":
		return 5
	case "telegram":
		return 6
	case "pushover":
		return 7
	case "pushbullet":
		return 8
	case "msteams", "ms teams", "microsoft teams":
		return 9
	case "zapier":
		return 11
	case "pagerduty", "pager duty":
		return 12
	case "googlechat", "google chat":
		return 13
	case "splunk":
		return 14
	case "mattermost":
		return 15
	default:
		return 2 // Default to webhook value
	}
}
