package websocket

import (
	"demodesk/neko/internal/types"
	"demodesk/neko/internal/types/event"
	"demodesk/neko/internal/types/message"
)

func (h *MessageHandler) controlRelease(session types.Session) error {
	// check if session is host
	if !h.sessions.IsHost(session.ID()) {
		h.logger.Debug().Str("id", session.ID()).Msg("is not the host")
		return nil
	}

	// release host
	h.logger.Debug().Str("id", session.ID()).Msgf("host called %s", event.CONTROL_RELEASE)
	h.sessions.ClearHost()

	// tell everyone
	if err := h.sessions.Broadcast(
		message.Control{
			Event: event.CONTROL_RELEASE,
			ID:    session.ID(),
		}, nil); err != nil {
		h.logger.Warn().Err(err).Msgf("broadcasting event %s has failed", event.CONTROL_RELEASE)
		return err
	}

	return nil
}

func (h *MessageHandler) controlRequest(session types.Session) error {
	// check for host
	if !h.sessions.HasHost() {
		// set host
		if err := h.sessions.SetHost(session.ID()); err != nil {
			h.logger.Warn().Err(err).Msgf("SetHost failed")
			return err
		}

		// let everyone know
		if err := h.sessions.Broadcast(
			message.Control{
				Event: event.CONTROL_LOCKED,
				ID:    session.ID(),
			}, nil); err != nil {
			h.logger.Warn().Err(err).Msgf("broadcasting event %s has failed", event.CONTROL_LOCKED)
			return err
		}

		return nil
	}

	// get host
	host, ok := h.sessions.GetHost()
	if ok {

		// tell session there is a host
		if err := session.Send(message.Control{
			Event: event.CONTROL_REQUEST,
			ID:    host.ID(),
		}); err != nil {
			h.logger.Warn().Err(err).Str("id", session.ID()).Msgf("sending event %s has failed", event.CONTROL_REQUEST)
			return err
		}

		// tell host session wants to be host
		if err := host.Send(message.Control{
			Event: event.CONTROL_REQUESTING,
			ID:    session.ID(),
		}); err != nil {
			h.logger.Warn().Err(err).Str("id", host.ID()).Msgf("sending event %s has failed", event.CONTROL_REQUESTING)
			return err
		}
	}

	return nil
}

func (h *MessageHandler) controlGive(session types.Session, payload *message.Control) error {
	// check if session is host
	if !h.sessions.IsHost(session.ID()) {
		h.logger.Debug().Str("id", session.ID()).Msg("is not the host")
		return nil
	}

	if !h.sessions.Has(payload.ID) {
		h.logger.Debug().Str("id", payload.ID).Msg("user does not exist")
		return nil
	}

	// set host
	if err := h.sessions.SetHost(payload.ID); err != nil {
		h.logger.Warn().Err(err).Msgf("SetHost failed")
		return err
	}

	// let everyone know
	if err := h.sessions.Broadcast(
		message.ControlTarget{
			Event:  event.CONTROL_GIVE,
			ID:     session.ID(),
			Target: payload.ID,
		}, nil); err != nil {
		h.logger.Warn().Err(err).Msgf("broadcasting event %s has failed", event.CONTROL_LOCKED)
		return err
	}

	return nil
}

func (h *MessageHandler) controlClipboard(session types.Session, payload *message.Clipboard) error {
	// check if session is host
	if !h.sessions.IsHost(session.ID()) {
		h.logger.Debug().Str("id", session.ID()).Msg("is not the host")
		return nil
	}

	h.remote.WriteClipboard(payload.Text)
	return nil
}

func (h *MessageHandler) controlKeyboard(session types.Session, payload *message.Keyboard) error {
	// check if session is host
	if !h.sessions.IsHost(session.ID()) {
		h.logger.Debug().Str("id", session.ID()).Msg("is not the host")
		return nil
	}

	// change layout
	if payload.Layout != nil {
		h.remote.SetKeyboardLayout(*payload.Layout)
	}

	// set num lock
	var NumLock = 0
	if payload.NumLock == nil {
		NumLock = -1
	} else if *payload.NumLock {
		NumLock = 1
	}

	// set caps lock
	var CapsLock = 0
	if payload.CapsLock == nil {
		CapsLock = -1
	} else if *payload.CapsLock {
		CapsLock = 1
	}

	// set scroll lock
	var ScrollLock = 0
	if payload.ScrollLock == nil {
		ScrollLock = -1
	} else if *payload.ScrollLock {
		ScrollLock = 1
	}

	h.logger.Debug().
		Int("NumLock", NumLock).
		Int("CapsLock", CapsLock).
		Int("ScrollLock", ScrollLock).
		Msg("setting keyboard modifiers")

	h.remote.SetKeyboardModifiers(NumLock, CapsLock, ScrollLock)
	return nil
}
