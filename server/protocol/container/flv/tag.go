package flv

import (
	"fmt"
)

type flvTag struct {
	fType     uint8
	dataSize  uint32
	timeStamp uint32
	streamID  uint32 // always 0
}

type mediaTag struct {
	/*
		SoundFormat: UB[4]
		0 = Linear PCM, platform endian
		1 = ADPCM
		2 = MP3
		3 = Linear PCM, little endian
		4 = Nellymoser 16-kHz mono
		5 = Nellymoser 8-kHz mono
		6 = Nellymoser
		7 = G.711 A-law logarithmic PCM
		8 = G.711 mu-law logarithmic PCM
		9 = reserved
		10 = AAC
		11 = Speex
		14 = MP3 8-Khz
		15 = Device-specific sound
		Formats 7, 8, 14, and 15 are reserved for errs use
		AAC is supported in Flash Player 9,0,115,0 and higher.
		Speex is supported in Flash Player 10 and higher.
	*/
	soundFormat SoundFormat

	/*
		SoundRate: UB[2]
		Sampling rate
		0 = 5.5-kHz For AAC: always 3
		1 = 11-kHz
		2 = 22-kHz
		3 = 44-kHz
	*/
	soundRate uint8

	/*
		SoundSize: UB[1]
		0 = snd8Bit
		1 = snd16Bit
		Size of each sample.
		This parameter only pertains to uncompressed formats.
		Compressed formats always decode to 16 bits internally
	*/
	soundSize uint8

	/*
		SoundType: UB[1]
		0 = sndMono
		1 = sndStereo
		Mono or stereo sound For Nellymoser: always 0
		For AAC: always 1
	*/
	soundType uint8

	/*
		0: AAC sequence header
		1: AAC raw
	*/
	aacPacketType uint8

	/*
		1: keyframe (for AVC, a seekable frame)
		2: inter frame (for AVC, a non- seekable frame)
		3: disposable inter frame (H.263 only)
		4: generated keyframe (reserved for server use only)
		5: video info/command frame
	*/
	frameType FrameType

	/*
		1: JPEG (currently unused)
		2: Sorenson H.263
		3: Screen video
		4: On2 VP6
		5: On2 VP6 with alpha channel
		6: Screen video version 2
		7: AVC
	*/
	codecID CodecID

	/*
		0: AVC sequence header
		1: AVC NALU
		2: AVC end of sequence (lower level NALU sequence ender is not required or supported)
	*/
	avcPacketType uint8

	compositionTime int32
}

type Tag struct {
	flvTag   flvTag
	mediaTag mediaTag
}

func (tag *Tag) Tag() string {
	return "flv"
}

func (tag *Tag) SoundFormat() SoundFormat {
	return tag.mediaTag.soundFormat
}

func (tag *Tag) AACPacketType() uint8 {
	return tag.mediaTag.aacPacketType
}

func (tag *Tag) IsKeyFrame() bool {
	return tag.mediaTag.frameType == Key
}

func (tag *Tag) IsSeq() bool {
	return tag.mediaTag.frameType == Key &&
		tag.mediaTag.avcPacketType == AVC_SEQHDR
}

func (tag *Tag) CodecID() CodecID {
	return tag.mediaTag.codecID
}

func (tag *Tag) CompositionTime() int32 {
	return tag.mediaTag.compositionTime
}

func (tag *Tag) ParseMediaTagHeader(b []byte, isVideo bool) (n int, err error) {
	switch isVideo {
	case false:
		n, err = tag.parseAudioHeader(b)
	case true:
		n, err = tag.parseVideoHeader(b)
	}
	return
}

func (tag *Tag) parseAudioHeader(b []byte) (n int, err error) {
	if len(b) < n+1 {
		err = fmt.Errorf("invalid audiodata len=%d", len(b))
		return
	}
	flags := b[0]
	tag.mediaTag.soundFormat = SoundFormat(flags >> 4)
	tag.mediaTag.soundRate = (flags >> 2) & 0x3
	tag.mediaTag.soundSize = (flags >> 1) & 0x1
	tag.mediaTag.soundType = flags & 0x1
	n++
	switch tag.mediaTag.soundFormat {
	case AAC:
		tag.mediaTag.aacPacketType = b[1]
		n++
	default:
		return -1, fmt.Errorf("not handle sound format")
	}
	return
}

func (tag *Tag) parseVideoHeader(b []byte) (n int, err error) {
	if len(b) < n+5 {
		err = fmt.Errorf("invalid videodata len=%d", len(b))
		return
	}
	flags := b[0]
	tag.mediaTag.frameType = FrameType(flags >> 4)
	tag.mediaTag.codecID = CodecID(flags & 0xf)
	n++
	if tag.mediaTag.frameType == Inter || tag.mediaTag.frameType == Key {
		tag.mediaTag.avcPacketType = b[1]
		for i := 2; i < 5; i++ {
			tag.mediaTag.compositionTime = tag.mediaTag.compositionTime<<8 + int32(b[i])
		}
		n += 4
	}
	return
}
