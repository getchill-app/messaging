package messaging

import (
	"time"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/encoding"
	"github.com/keys-pub/keys/tsutil"
)

// Message is encrypted by clients.
type Message struct {
	ID        string  `json:"id" msgpack:"id" db:"id"`
	Channel   keys.ID `json:"channel" msgpack:"channel" db:"channel"`
	Sender    keys.ID `json:"sender" msgpack:"sender" db:"sender"`
	Timestamp int64   `json:"ts,omitempty" msgpack:"ts,omitempty" db:"ts"`
	Prev      string  `json:"prev,omitempty" msgpack:"prev,omitempty" db:"prev"`

	// For message text (optional).
	Text string `json:"text,omitempty" msgpack:"text,omitempty" db:"text"`

	// Command encodes other types of messages.
	Command *MessageCommand `json:"cmd,omitempty" msgpack:"cmd,omitempty" db:"cmd"`

	// RemoteIndex is set from the remote events API (untrusted).
	RemoteIndex int64 `json:"-" msgpack:"-" db:"ridx"`
	// RemoteTimestamp is set from the remote events API (untrusted).
	RemoteTimestamp int64 `json:"-" msgpack:"-" db:"rts"`
}

// NewID returns a new random ID (string).
func NewID() string {
	return encoding.MustEncode(keys.RandBytes(32), encoding.Base62)
}

// NewMessage creates a new empty message.
func NewMessage(channel keys.ID, sender keys.ID) *Message {
	return &Message{
		ID:        NewID(),
		Channel:   channel,
		Sender:    sender,
		Timestamp: tsutil.Millis(time.Now()),
	}
}

// WithPrev ...
func (m *Message) WithPrev(prev string) *Message {
	m.Prev = prev
	return m
}

// WithText ...
func (m *Message) WithText(text string) *Message {
	m.Text = text
	return m
}

// WithTimestamp ...
func (m *Message) WithTimestamp(ts int64) *Message {
	m.Timestamp = ts
	return m
}
