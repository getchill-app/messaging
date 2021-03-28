package messaging

import (
	"database/sql/driver"
	"time"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/keys-pub/keys/encoding"
	"github.com/keys-pub/keys/tsutil"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v4"
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

// Encrypt message.
func (m *Message) Encrypt(sender *keys.EdX25519Key, recipient keys.ID) ([]byte, error) {
	if m.RemoteTimestamp != 0 {
		return nil, errors.Errorf("remote timestamp should be omitted on send")
	}
	if m.RemoteIndex != 0 {
		return nil, errors.Errorf("remote index should be omitted on send")
	}
	if m.Timestamp == 0 {
		return nil, errors.Errorf("message timestamp is not set")
	}
	if m.Sender == "" {
		return nil, errors.Errorf("message sender not set")
	}
	if m.Sender != sender.ID() {
		return nil, errors.Errorf("message sender mismatch")
	}
	return Encrypt(m, sender, recipient)
}

// MessageCommand encodes other types of messages.
type MessageCommand struct {
	// ChannelInfo sets info.
	ChannelInfo *ChannelInfo `json:"channelInfo,omitempty" msgpack:"channelInfo,omitempty"`
	// ChannelInvites to invite to a new channel.
	ChannelInvites []*ChannelInvite `json:"channelInvites,omitempty" msgpack:"channelInvites,omitempty"`
}

// ChannelInfo for setting channel name or description.
type ChannelInfo struct {
	Name        string `json:"name,omitempty" msgpack:"name,omitempty"`
	Description string `json:"desc,omitempty" msgpack:"desc,omitempty"`
}

// NewMessageForChannelInfo ...
func NewMessageForChannelInfo(channel keys.ID, sender keys.ID, info *ChannelInfo) *Message {
	msg := NewMessage(channel, sender)
	msg.Command = &MessageCommand{ChannelInfo: info}
	return msg
}

// ChannelInvite if invited to a channel.
type ChannelInvite struct {
	Channel   keys.ID      `json:"channel" msgpack:"channel"`
	Recipient keys.ID      `json:"recipient" msgpack:"recipient"`
	Sender    keys.ID      `json:"sender" msgpack:"sender"`
	Key       *api.Key     `json:"key" msgpack:"key"`
	Token     string       `json:"token" msgpack:"token"`
	Info      *ChannelInfo `json:"info,omitempty" msgpack:"info,omitempty"`
}

// NewMessageForChannelInvites ...
func NewMessageForChannelInvites(channel keys.ID, sender keys.ID, invites []*ChannelInvite) *Message {
	msg := NewMessage(channel, sender)
	msg.Command = &MessageCommand{ChannelInvites: invites}
	return msg
}

// Scan for sql.DB.
func (i *MessageCommand) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		if len(v) == 0 {
			return nil
		}
		var cmd MessageCommand
		if err := msgpack.Unmarshal([]byte(v), &cmd); err != nil {
			return err
		}
		*i = cmd
		return nil
	default:
		return errors.Errorf("invalid db type for MessageCommand: %T", src)
	}
}

// Value for sql.DB.
func (i *MessageCommand) Value() (driver.Value, error) {
	if i == nil {
		return driver.Value(nil), nil
	}

	b, err := msgpack.Marshal(i)
	if err != nil {
		return nil, err
	}
	return driver.Value(b), nil
}

// Encrypt does crypto_box_seal(pk+crypto_box(msgpack(i))).
func Encrypt(i interface{}, sender *keys.EdX25519Key, recipient keys.ID) ([]byte, error) {
	pk := api.NewKey(recipient).AsX25519Public()
	if pk == nil {
		return nil, errors.Errorf("invalid message recipient")
	}
	b, err := msgpack.Marshal(i)
	if err != nil {
		return nil, err
	}
	sk := sender.X25519Key()
	encrypted := keys.BoxSeal(b, pk, sk)
	box := append(sk.Public(), encrypted...)
	anonymized := keys.CryptoBoxSeal(box, pk)
	return anonymized, nil
}

// DecryptMessage decrypts message.
func DecryptMessage(b []byte, key *keys.EdX25519Key) (*Message, error) {
	var message Message
	pk, err := Decrypt(b, &message, key)
	if err != nil {
		return nil, err
	}
	expected := api.NewKey(message.Sender).AsX25519Public()
	if pk.ID() != expected.ID() {
		return nil, errors.Errorf("message sender mismatch")
	}
	return &message, nil
}

// Decrypt value, returning sender public key.
func Decrypt(b []byte, v interface{}, key *keys.EdX25519Key) (*keys.X25519PublicKey, error) {
	box, err := keys.CryptoBoxSealOpen(b, key.X25519Key())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decrypt message")
	}
	if len(box) < 32 {
		return nil, errors.Wrapf(errors.Errorf("not enough bytes"), "failed to decrypt message")
	}
	pk := keys.NewX25519PublicKey(keys.Bytes32(box[:32]))
	encrypted := box[32:]

	decrypted, err := keys.BoxOpen(encrypted, pk, key.X25519Key())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decrypt message")
	}

	if err := msgpack.Unmarshal(decrypted, v); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal message")
	}
	return pk, nil
}
