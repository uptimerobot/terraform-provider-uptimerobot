package provider

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
	}
}
