package constdef

type SessionConfigKey string

const (
	ConfigKeyStreamBase SessionConfigKey = "stream_base"
	ConfigKeyStreamSess SessionConfigKey = "stream_sess"

	ConfigKeySessionBase SessionConfigKey = "session_base"
	ConfigKeyVideoCodec  SessionConfigKey = "video_codec"
	ConfigKeyAudioCodec  SessionConfigKey = "audio_codec"

	ConfigKeySessionType SessionConfigKey = "session_type"

	ConfigKeySinkSourceSession SessionConfigKey = "config_sink_source_session"
)
