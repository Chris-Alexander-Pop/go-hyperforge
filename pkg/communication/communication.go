package communication

// Channel identifies a communication capability.
type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelPush     Channel = "push"
	ChannelChat     Channel = "chat"
	ChannelTemplate Channel = "template"
)

// Driver constants for all communication backends.
const (
	// Shared / testing
	DriverMemory = "memory"

	// Email drivers
	DriverSendGrid = "sendgrid"
	DriverSES      = "ses"
	DriverSMTP     = "smtp"

	// SMS drivers
	DriverTwilio = "twilio"
	DriverSNS    = "sns"

	// Push drivers
	DriverFCM  = "fcm"
	DriverAPNS = "apns"

	// Chat drivers
	DriverSlack   = "slack"
	DriverDiscord = "discord"

	// Template drivers
	DriverText = "text"
	DriverHTML = "html"
)
