// Auto-generated xd
package twitch

type EventHandler struct {
	onClearChatMessage []func(message ClearChatMessage)

	onNamesMessage []func(message NamesMessage)

	onNoticeMessage []func(message NoticeMessage)

	onPingMessage []func(message PingMessage)

	onPongMessage []func(message PongMessage)

	onPrivateMessage []func(message PrivateMessage)

	onRawMessage []func(message RawMessage)

	onReconnectMessage []func(message ReconnectMessage)

	onRoomStateMessage []func(message RoomStateMessage)

	onUserJoinMessage []func(message UserJoinMessage)

	onUserNoticeMessage []func(message UserNoticeMessage)

	onUserPartMessage []func(message UserPartMessage)

	onUserStateMessage []func(message UserStateMessage)

	onWhisperMessage []func(message WhisperMessage)
}

func (e *Client) handleMessage(message Message) (err error) {
	switch msg := message.(type) {

	case *ClearChatMessage:

		for _, cb := range e.onClearChatMessage {
			cb(*msg)
		}

	case *NamesMessage:

		for _, cb := range e.onNamesMessage {
			cb(*msg)
		}

		err = e.handleNamesMessage(*msg)

	case *NoticeMessage:

		for _, cb := range e.onNoticeMessage {
			cb(*msg)
		}

		err = e.handleNoticeMessage(*msg)

	case *PingMessage:

		for _, cb := range e.onPingMessage {
			cb(*msg)
		}

		err = e.handlePingMessage(*msg)

	case *PongMessage:

		for _, cb := range e.onPongMessage {
			cb(*msg)
		}

		err = e.handlePongMessage(*msg)

	case *PrivateMessage:

		for _, cb := range e.onPrivateMessage {
			cb(*msg)
		}

	case *RawMessage:

		for _, cb := range e.onRawMessage {
			cb(*msg)
		}

	case *ReconnectMessage:

		for _, cb := range e.onReconnectMessage {
			cb(*msg)
		}

		err = e.handleReconnectMessage(*msg)

	case *RoomStateMessage:

		for _, cb := range e.onRoomStateMessage {
			cb(*msg)
		}

	case *UserJoinMessage:

		if !e.handleUserJoinMessage(*msg) {
			return nil
		}

		for _, cb := range e.onUserJoinMessage {
			cb(*msg)
		}

	case *UserNoticeMessage:

		for _, cb := range e.onUserNoticeMessage {
			cb(*msg)
		}

	case *UserPartMessage:

		if !e.handleUserPartMessage(*msg) {
			return nil
		}

		for _, cb := range e.onUserPartMessage {
			cb(*msg)
		}

	case *UserStateMessage:

		for _, cb := range e.onUserStateMessage {
			cb(*msg)
		}

	case *WhisperMessage:

		for _, cb := range e.onWhisperMessage {
			cb(*msg)
		}

	}

	return
}

// OnClearChatMessage test
func (e *EventHandler) OnClearChatMessage(cb func(ClearChatMessage)) {
	e.onClearChatMessage = append(e.onClearChatMessage, cb)
}

// OnNamesMessage test
func (e *EventHandler) OnNamesMessage(cb func(NamesMessage)) {
	e.onNamesMessage = append(e.onNamesMessage, cb)
}

// OnNoticeMessage test
func (e *EventHandler) OnNoticeMessage(cb func(NoticeMessage)) {
	e.onNoticeMessage = append(e.onNoticeMessage, cb)
}

// OnPingMessage test
func (e *EventHandler) OnPingMessage(cb func(PingMessage)) {
	e.onPingMessage = append(e.onPingMessage, cb)
}

// OnPongMessage test
func (e *EventHandler) OnPongMessage(cb func(PongMessage)) {
	e.onPongMessage = append(e.onPongMessage, cb)
}

// OnPrivateMessage test
func (e *EventHandler) OnPrivateMessage(cb func(PrivateMessage)) {
	e.onPrivateMessage = append(e.onPrivateMessage, cb)
}

// OnRawMessage test
func (e *EventHandler) OnRawMessage(cb func(RawMessage)) {
	e.onRawMessage = append(e.onRawMessage, cb)
}

// OnReconnectMessage test
func (e *EventHandler) OnReconnectMessage(cb func(ReconnectMessage)) {
	e.onReconnectMessage = append(e.onReconnectMessage, cb)
}

// OnRoomStateMessage test
func (e *EventHandler) OnRoomStateMessage(cb func(RoomStateMessage)) {
	e.onRoomStateMessage = append(e.onRoomStateMessage, cb)
}

// OnUserJoinMessage test
func (e *EventHandler) OnUserJoinMessage(cb func(UserJoinMessage)) {
	e.onUserJoinMessage = append(e.onUserJoinMessage, cb)
}

// OnUserNoticeMessage test
func (e *EventHandler) OnUserNoticeMessage(cb func(UserNoticeMessage)) {
	e.onUserNoticeMessage = append(e.onUserNoticeMessage, cb)
}

// OnUserPartMessage test
func (e *EventHandler) OnUserPartMessage(cb func(UserPartMessage)) {
	e.onUserPartMessage = append(e.onUserPartMessage, cb)
}

// OnUserStateMessage test
func (e *EventHandler) OnUserStateMessage(cb func(UserStateMessage)) {
	e.onUserStateMessage = append(e.onUserStateMessage, cb)
}

// OnWhisperMessage test
func (e *EventHandler) OnWhisperMessage(cb func(WhisperMessage)) {
	e.onWhisperMessage = append(e.onWhisperMessage, cb)
}
