package constdef

type SessionConfigKey string

const (
	ConfigKeyStreamBase SessionConfigKey = "ConfigKeyStreamBase"
	ConfigKeyStreamSess SessionConfigKey = "ConfigKeyStreamSess"

	ConfigKeySessionBase SessionConfigKey = "ConfigKeySessionBase"
	ConfigKeyVideoCodec  SessionConfigKey = "ConfigKeyVideoCodec"
	ConfigKeyAudioCodec  SessionConfigKey = "ConfigKeyAudioCodec"

	ConfigKeySessionType SessionConfigKey = "ConfigKeySinkSourceSession"

	ConfigKeySinkSourceSession SessionConfigKey = "ConfigKeySinkSourceSession"

	ConfigKeySinkRW SessionConfigKey = "ConfigKeySinkRW"
)
