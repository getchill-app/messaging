package messaging

import (
	"github.com/keys-pub/keys"
)

type ChannelStatus struct {
	Channel         keys.ID `json:"channel" msgpack:"channel" db:"channel"`
	Name            string  `json:"name,omitempty" msgpack:"name,omitempty" db:"name"`
	Description     string  `json:"desc,omitempty" msgpack:"desc,omitempty" db:"desc"`
	Snippet         string  `json:"snippet,omitempty" msgpack:"snippet,omitempty" db:"snippet"`
	Index           int64   `json:"index,omitempty" msgpack:"index,omitempty" db:"index"`
	Timestamp       int64   `json:"ts,omitempty" msgpack:"ts,omitempty" db:"ts"`
	RemoteTimestamp int64   `json:"rts,omitempty" msgpack:"rts,omitempty" db:"rts"`
	ReadIndex       int64   `json:"readIndex,omitempty" msgpack:"readIndex,omitempty" db:"readIndex"`
}

func (c *ChannelStatus) Update(msg *Message) {
	if len(msg.Text) > 0 {
		c.Snippet = msg.Text
	}

	// Update channel info
	if msg.Command != nil {
		if msg.Command.ChannelInfo != nil {
			if msg.Command.ChannelInfo.Name != "" {
				c.Name = msg.Command.ChannelInfo.Name
			}
			if msg.Command.ChannelInfo.Description != "" {
				c.Description = msg.Command.ChannelInfo.Description
			}
		}
	}
	c.Timestamp = msg.Timestamp
	c.RemoteTimestamp = msg.RemoteTimestamp
}
