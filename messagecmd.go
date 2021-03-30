package messaging

import (
	"database/sql/driver"

	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v4"
)

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
