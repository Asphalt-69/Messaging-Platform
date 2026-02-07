package protocol

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Message types
const (
	TypeAuth          = "auth"
	TypeMessage       = "message"
	TypeGroupMessage  = "group_message"
	TypeTyping        = "typing"
	TypePresence      = "presence"
	TypeACK           = "ack"
	TypeError         = "error"
	TypeHeartbeat     = "heartbeat"
)

// BaseMessage is the common structure for all messages
type BaseMessage struct {
	Type      string    `json:"type"`
	MessageID string    `json:"message_id,omitempty"`
	Timestamp int64     `json:"timestamp"`
}

// AuthMessage for authentication
type AuthMessage struct {
	BaseMessage
	Token string `json:"token"`
}

// TextMessage for direct messaging
type TextMessage struct {
	BaseMessage
	From    string          `json:"from"`
	To      string          `json:"to"`
	Payload TextPayload     `json:"payload"`
}

type TextPayload struct {
	Text      string            `json:"text"`
	MediaURL  string            `json:"media_url,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	ReplyTo   string            `json:"reply_to,omitempty"`
}

// GroupMessage for group chats
type GroupMessage struct {
	BaseMessage
	From      string          `json:"from"`
	GroupID   string          `json:"group_id"`
	Payload   TextPayload     `json:"payload"`
}

// TypingIndicator for real-time typing events
type TypingIndicator struct {
	BaseMessage
	UserID    string `json:"user_id"`
	ChatID    string `json:"chat_id"`
	IsTyping  bool   `json:"is_typing"`
}

// PresenceUpdate for online/offline status
type PresenceUpdate struct {
	BaseMessage
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"` // online, away, offline
	LastSeen  int64     `json:"last_seen,omitempty"`
	Device    string    `json:"device,omitempty"`
}

// Acknowledgement message
type Acknowledgement struct {
	BaseMessage
	OriginalMessageID string `json:"original_message_id"`
	Status            string `json:"status"` // delivered, read, failed
}

// ErrorMessage for error responses
type ErrorMessage struct {
	BaseMessage
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Heartbeat for keepalive
type Heartbeat struct {
	BaseMessage
	Sequence int64 `json:"sequence"`
}

// ParseMessage parses raw JSON into appropriate message type
func ParseMessage(raw []byte) (interface{}, error) {
	var base BaseMessage
	if err := json.Unmarshal(raw, &base); err != nil {
		return nil, fmt.Errorf("invalid message format: %w", err)
	}

	switch base.Type {
	case TypeAuth:
		var msg AuthMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case TypeMessage:
		var msg TextMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case TypeGroupMessage:
		var msg GroupMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case TypeTyping:
		var msg TypingIndicator
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case TypePresence:
		var msg PresenceUpdate
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case TypeACK:
		var msg Acknowledgement
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case TypeHeartbeat:
		var msg Heartbeat
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", base.Type)
	}
}

// NewBaseMessage creates a new base message with ID and timestamp
func NewBaseMessage(msgType string) BaseMessage {
	return BaseMessage{
		Type:      msgType,
		MessageID: uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewErrorMessage creates a new error message
func NewErrorMessage(code, message, details string) ErrorMessage {
	return ErrorMessage{
		BaseMessage: NewBaseMessage(TypeError),
		Code:        code,
		Message:     message,
		Details:     details,
	}
}
