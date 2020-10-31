package websocket

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"demodesk/neko/internal/types"
	"demodesk/neko/internal/types/event"
	"demodesk/neko/internal/types/message"
	"demodesk/neko/internal/utils"
)

type MessageHandler struct {
	logger    zerolog.Logger
	sessions  types.SessionManager
	webrtc    types.WebRTCManager
	remote    types.RemoteManager
	broadcast types.BroadcastManager
	banned    map[string]bool
	locked    bool
}

func (h *MessageHandler) Connected(id string, socket *WebSocket) (bool, string, error) {
	address := socket.Address()
	if address == "" {
		h.logger.Debug().Msg("no remote address")
	} else {
		ok, banned := h.banned[address]
		if ok && banned {
			h.logger.Debug().Str("address", address).Msg("banned")
			return false, "banned", nil
		}
	}

	if h.locked {
		session, ok := h.sessions.Get(id)
		if !ok || !session.Admin() {
			h.logger.Debug().Msg("server locked")
			return false, "locked", nil
		}
	}

	return true, "", nil
}

func (h *MessageHandler) Disconnected(id string) error {
	if h.locked && len(h.sessions.Admins()) == 0 {
		h.locked = false
	}

	return h.sessions.Destroy(id)
}

func (h *MessageHandler) Message(id string, raw []byte) error {
	header := message.Message{}
	if err := json.Unmarshal(raw, &header); err != nil {
		return err
	}

	session, ok := h.sessions.Get(id)
	if !ok {
		return errors.Errorf("unknown session id %s", id)
	}

	switch header.Event {
	// Signal Events
	case event.SIGNAL_ANSWER:
		payload := &message.SignalAnswer{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.signalAnswer(session, payload)
			}), "%s failed", header.Event)

	// Control Events
	case event.CONTROL_RELEASE:
		return errors.Wrapf(h.controlRelease(session), "%s failed", header.Event)
	case event.CONTROL_REQUEST:
		return errors.Wrapf(h.controlRequest(session), "%s failed", header.Event)
	case event.CONTROL_GIVE:
		payload := &message.Control{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.controlGive(session, payload)
			}), "%s failed", header.Event)
	case event.CONTROL_CLIPBOARD:
		payload := &message.Clipboard{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.controlClipboard(session, payload)
			}), "%s failed", header.Event)
	case event.CONTROL_KEYBOARD:
		payload := &message.Keyboard{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.controlKeyboard(session, payload)
			}), "%s failed", header.Event)

	// Screen Events
	case event.SCREEN_RESOLUTION:
		return errors.Wrapf(h.screenResolution(session), "%s failed", header.Event)
	case event.SCREEN_CONFIGURATIONS:
		return errors.Wrapf(h.screenConfigurations(session), "%s failed", header.Event)
	case event.SCREEN_SET:
		payload := &message.ScreenResolution{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.screenSet(session, payload)
			}), "%s failed", header.Event)

	// Boradcast Events
	case event.BORADCAST_CREATE:
		payload := &message.BroadcastCreate{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.boradcastCreate(session, payload)
			}), "%s failed", header.Event)
	case event.BORADCAST_DESTROY:
		return errors.Wrapf(h.boradcastDestroy(session), "%s failed", header.Event)

	// Admin Events
	case event.ADMIN_LOCK:
		return errors.Wrapf(h.adminLock(session), "%s failed", header.Event)
	case event.ADMIN_UNLOCK:
		return errors.Wrapf(h.adminUnlock(session), "%s failed", header.Event)
	case event.ADMIN_CONTROL:
		return errors.Wrapf(h.adminControl(session), "%s failed", header.Event)
	case event.ADMIN_RELEASE:
		return errors.Wrapf(h.adminRelease(session), "%s failed", header.Event)
	case event.ADMIN_GIVE:
		payload := &message.Admin{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.adminGive(session, payload)
			}), "%s failed", header.Event)
	case event.ADMIN_BAN:
		payload := &message.Admin{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.adminBan(session, payload)
			}), "%s failed", header.Event)
	case event.ADMIN_KICK:
		payload := &message.Admin{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.adminKick(session, payload)
			}), "%s failed", header.Event)
	case event.ADMIN_MUTE:
		payload := &message.Admin{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.adminMute(session, payload)
			}), "%s failed", header.Event)
	case event.ADMIN_UNMUTE:
		payload := &message.Admin{}
		return errors.Wrapf(
			utils.Unmarshal(payload, raw, func() error {
				return h.adminUnmute(session, payload)
			}), "%s failed", header.Event)
	default:
		return errors.Errorf("unknown message event %s", header.Event)
	}
}
