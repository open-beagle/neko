package capture

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"demodesk/neko/internal/config"
	"demodesk/neko/internal/types"
	"demodesk/neko/internal/types/codec"
)

type CaptureManagerCtx struct {
	logger  zerolog.Logger
	desktop types.DesktopManager

	// sinks
	broadcast  *BroacastManagerCtx
	screencast *ScreencastManagerCtx
	audio      *StreamSinkManagerCtx
	videos     map[string]*StreamSinkManagerCtx
	videoIDs   []string

	// sources
	webcam     *StreamSrcManagerCtx
	microphone *StreamSrcManagerCtx
}

func New(desktop types.DesktopManager, config *config.Capture) *CaptureManagerCtx {
	logger := log.With().Str("module", "capture").Logger()

	broadcastPipeline := config.BroadcastPipeline
	if broadcastPipeline == "" {
		broadcastPipeline = fmt.Sprintf(
			"flvmux name=mux ! rtmpsink location='{url} live=1' "+
				"pulsesrc device=%s "+
				"! audio/x-raw,channels=2 "+
				"! audioconvert "+
				"! queue "+
				"! voaacenc bitrate=%d "+
				"! mux. "+
				"ximagesrc display-name=%s show-pointer=true use-damage=false "+
				"! video/x-raw "+
				"! videoconvert "+
				"! queue "+
				"! x264enc threads=4 bitrate=%d key-int-max=15 byte-stream=true tune=zerolatency speed-preset=%s "+
				"! mux.", config.AudioDevice, config.BroadcastAudioBitrate*1000, config.Display, config.BroadcastVideoBitrate, config.BroadcastPreset,
		)
	}

	screencastPipeline := config.ScreencastPipeline
	if screencastPipeline == "" {
		screencastPipeline = fmt.Sprintf(
			"ximagesrc display-name=%s show-pointer=true use-damage=false "+
				"! video/x-raw,framerate=%s "+
				"! videoconvert "+
				"! queue "+
				"! jpegenc quality=%s "+
				"! appsink name=appsink", config.Display, config.ScreencastRate, config.ScreencastQuality,
		)
	}

	videos := map[string]*StreamSinkManagerCtx{}
	for video_id, cnf := range config.VideoPipelines {
		pipelineConf := cnf

		createPipeline := func() string {
			if pipelineConf.GstPipeline != "" {
				return strings.Replace(pipelineConf.GstPipeline, "{display}", config.Display, 1)
			}

			screen := desktop.GetScreenSize()
			pipeline, err := pipelineConf.GetPipeline(*screen)
			if err != nil {
				logger.Panic().Err(err).
					Str("video_id", video_id).
					Msg("unable to get video pipeline")
			}

			return fmt.Sprintf(
				"ximagesrc display-name=%s show-pointer=false use-damage=false "+
					"%s ! appsink name=appsink", config.Display, pipeline,
			)
		}

		// trigger function to catch evaluation errors at startup
		pipeline := createPipeline()
		logger.Info().
			Str("video_id", video_id).
			Str("pipeline", pipeline).
			Msg("syntax check for video stream pipeline passed")

		// append to videos
		videos[video_id] = streamSinkNew(config.VideoCodec, createPipeline, video_id)
	}

	return &CaptureManagerCtx{
		logger:  logger,
		desktop: desktop,

		// sinks
		broadcast:  broadcastNew(broadcastPipeline),
		screencast: screencastNew(config.ScreencastEnabled, screencastPipeline),
		audio: streamSinkNew(config.AudioCodec, func() string {
			if config.AudioPipeline != "" {
				return config.AudioPipeline
			}

			return fmt.Sprintf(
				"pulsesrc device=%s "+
					"! audio/x-raw,channels=2 "+
					"! audioconvert "+
					"! queue "+
					"! %s "+
					"! appsink name=appsink", config.AudioDevice, config.AudioCodec.Pipeline,
			)
		}, "audio"),
		videos:   videos,
		videoIDs: config.VideoIDs,

		// sources
		webcam: streamSrcNew(map[string]string{
			codec.VP8().Name: "appsrc format=time is-live=true do-timestamp=true name=src " +
				fmt.Sprintf("! application/x-rtp, payload=%d, encoding-name=VP8-DRAFT-IETF-01 ", codec.VP8().PayloadType) +
				"! rtpvp8depay " +
				"! decodebin " +
				"! videoconvert " +
				"! v4l2sink device=/dev/video0",
			codec.VP9().Name: "appsrc format=time is-live=true do-timestamp=true name=src " +
				"! application/x-rtp " +
				"! rtpvp9depay " +
				"! decodebin " +
				"! videoconvert " +
				"! v4l2sink device=/dev/video0",
			codec.H264().Name: "appsrc format=time is-live=true do-timestamp=true name=src " +
				"! application/x-rtp " +
				"! rtph264depay " +
				"! decodebin " +
				"! videoconvert " +
				"! v4l2sink device=/dev/video0",
		}, "webcam"),
		microphone: streamSrcNew(map[string]string{
			codec.Opus().Name: "appsrc format=time is-live=true do-timestamp=true name=src " +
				fmt.Sprintf("! application/x-rtp, payload=%d, encoding-name=OPUS ", codec.Opus().PayloadType) +
				"! rtpopusdepay " +
				"! decodebin " +
				"! pulsesink device=audio_input",
			codec.G722().Name: "appsrc format=time is-live=true do-timestamp=true name=src " +
				"! application/x-rtp clock-rate=8000 " +
				"! rtpg722depay " +
				"! decodebin " +
				"! pulsesink device=audio_input",
		}, "microphone"),
	}
}

func (manager *CaptureManagerCtx) Start() {
	if manager.broadcast.Started() {
		if err := manager.broadcast.createPipeline(); err != nil {
			manager.logger.Panic().Err(err).Msg("unable to create broadcast pipeline")
		}
	}

	manager.desktop.OnBeforeScreenSizeChange(func() {
		for _, video := range manager.videos {
			if video.Started() {
				video.destroyPipeline()
			}
		}

		if manager.broadcast.Started() {
			manager.broadcast.destroyPipeline()
		}

		if manager.screencast.Started() {
			manager.screencast.destroyPipeline()
		}
	})

	manager.desktop.OnAfterScreenSizeChange(func() {
		for _, video := range manager.videos {
			if video.Started() {
				err := video.createPipeline()
				if err != nil && !errors.Is(err, types.ErrCapturePipelineAlreadyExists) {
					manager.logger.Panic().Err(err).Msg("unable to recreate video pipeline")
				}
			}
		}

		if manager.broadcast.Started() {
			err := manager.broadcast.createPipeline()
			if err != nil && !errors.Is(err, types.ErrCapturePipelineAlreadyExists) {
				manager.logger.Panic().Err(err).Msg("unable to recreate broadcast pipeline")
			}
		}

		if manager.screencast.Started() {
			err := manager.screencast.createPipeline()
			if err != nil && !errors.Is(err, types.ErrCapturePipelineAlreadyExists) {
				manager.logger.Panic().Err(err).Msg("unable to recreate screencast pipeline")
			}
		}
	})
}

func (manager *CaptureManagerCtx) Shutdown() error {
	manager.logger.Info().Msgf("shutdown")

	manager.broadcast.shutdown()
	manager.screencast.shutdown()

	manager.audio.shutdown()

	for _, video := range manager.videos {
		video.shutdown()
	}

	manager.webcam.shutdown()
	manager.microphone.shutdown()

	return nil
}

func (manager *CaptureManagerCtx) Broadcast() types.BroadcastManager {
	return manager.broadcast
}

func (manager *CaptureManagerCtx) Screencast() types.ScreencastManager {
	return manager.screencast
}

func (manager *CaptureManagerCtx) Audio() types.StreamSinkManager {
	return manager.audio
}

func (manager *CaptureManagerCtx) Video(videoID string) (types.StreamSinkManager, bool) {
	video, ok := manager.videos[videoID]
	return video, ok
}

func (manager *CaptureManagerCtx) VideoIDs() []string {
	return manager.videoIDs
}

func (manager *CaptureManagerCtx) Webcam() types.StreamSrcManager {
	return manager.webcam
}

func (manager *CaptureManagerCtx) Microphone() types.StreamSrcManager {
	return manager.microphone
}
