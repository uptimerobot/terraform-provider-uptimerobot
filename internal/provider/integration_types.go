package provider

import "strings"

// IntegrationType represents the supported integration types.
type IntegrationType string

const (
	IntegrationTypeSlack      IntegrationType = "slack"
	IntegrationTypeEmail      IntegrationType = "email"
	IntegrationTypeWebhook    IntegrationType = "webhook"
	IntegrationTypeSMS        IntegrationType = "sms"
	IntegrationTypeDiscord    IntegrationType = "discord"
	IntegrationTypeTelegram   IntegrationType = "telegram"
	IntegrationTypePushover   IntegrationType = "pushover"
	IntegrationTypePushbullet IntegrationType = "pushbullet"
	IntegrationTypeMSTeams    IntegrationType = "msteams"
	IntegrationTypeZapier     IntegrationType = "zapier"
	IntegrationTypePagerDuty  IntegrationType = "pagerduty"
	IntegrationTypeGoogleChat IntegrationType = "googlechat"
	IntegrationTypeSplunk     IntegrationType = "splunk"
)

// AllIntegrationTypes returns a slice of all supported integration types.
func AllIntegrationTypes() []string {
	return []string{
		string(IntegrationTypeSlack),
		string(IntegrationTypeEmail),
		string(IntegrationTypeWebhook),
		string(IntegrationTypeSMS),
		string(IntegrationTypeDiscord),
		string(IntegrationTypeTelegram),
		string(IntegrationTypePushover),
		string(IntegrationTypePushbullet),
		string(IntegrationTypeMSTeams),
		string(IntegrationTypeZapier),
		string(IntegrationTypePagerDuty),
		string(IntegrationTypeGoogleChat),
		string(IntegrationTypeSplunk),
	}
}

// IntegrationTypeDescriptions returns a map of integration types to their descriptions.
func IntegrationTypeDescriptions() map[string]string {
	return map[string]string{
		string(IntegrationTypeSlack):      "Slack webhook integration",
		string(IntegrationTypeEmail):      "Email notifications",
		string(IntegrationTypeWebhook):    "Custom webhook integration",
		string(IntegrationTypeSMS):        "SMS notifications",
		string(IntegrationTypeDiscord):    "Discord webhook integration",
		string(IntegrationTypeTelegram):   "Telegram bot integration",
		string(IntegrationTypePushover):   "Pushover notifications",
		string(IntegrationTypePushbullet): "Pushbullet notifications",
		string(IntegrationTypeMSTeams):    "Microsoft Teams integration",
		string(IntegrationTypeZapier):     "Zapier integration",
		string(IntegrationTypePagerDuty):  "PagerDuty integration",
		string(IntegrationTypeGoogleChat): "Google Chat integration",
		string(IntegrationTypeSplunk):     "Splunk integration",
	}
}

// TransformIntegrationTypeToAPI converts case-insensitive terraform values to API format.
func TransformIntegrationTypeToAPI(terraformType string) string {
	switch strings.ToLower(terraformType) {
	case "slack":
		return "Slack"
	case "email":
		return "Email"
	case "webhook":
		return "Webhook"
	case "sms":
		return "SMS"
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
	default:
		return terraformType
	}
}

// TransformIntegrationTypeFromAPI converts API format to terraform format.
func TransformIntegrationTypeFromAPI(apiType string) string {
	switch apiType {
	case "Slack":
		return "slack"
	case "Email":
		return "email"
	case "Webhook":
		return "webhook"
	case "SMS":
		return "sms"
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
	default:
		return strings.ToLower(apiType)
	}
}
