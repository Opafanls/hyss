package flv

type SoundFormat uint8

const (
	PCMPlatformEndian SoundFormat = iota
	AD_PCM
	MP3
	PCMLittleEndian
	Nelly16K
	Nelly8K
	Nelly
	G711ALowPCM
	G711MULowPCM
	Reserved
	AAC
	Speex
	MP38K
	DeviceSpecific
)

type FrameType uint8

const (
	Key FrameType = iota
	Inter
	DisposableInter
	GeneratedKey
	VideoInfoCommand
)

type CodecID uint8

const (
	JPEG CodecID = iota
	H263
	Screen
	ON2VP6
	ON2VP6Alpha
	ScreenVideoV2
	H264
)

const (
	AVC_SEQHDR         = 0
	Tag_Audio          = 0x08
	Tag_Video          = 0x09
	Tag_Script         = 0x12
	Tag_ScriptDataAMF3 = 0xf
)
