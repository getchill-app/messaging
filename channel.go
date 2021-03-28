package messaging

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"
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

func updateChannelStatusTx(tx *sqlx.Tx, channel *ChannelStatus) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO channelStatus (channel, name, desc, snippet, "index", readIndex, ts, rts) VALUES 
		($1, $2, $3, $4, $5, $6, $7, $8);`,
		channel.Channel,
		channel.Name,
		channel.Description,
		channel.Snippet,
		channel.Index,
		channel.ReadIndex,
		channel.Timestamp,
		channel.RemoteTimestamp); err != nil {
		return errors.Wrapf(err, "failed to update channel status")
	}
	return nil
}

func getChannelStatus(db *sqlx.DB, channel keys.ID) (*ChannelStatus, error) {
	var channelStatus ChannelStatus
	if err := db.Get(&channelStatus, "SELECT * from channelStatus WHERE channel = ?", channel); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &channelStatus, nil
}

func getChannelStatuses(db *sqlx.DB) ([]*ChannelStatus, error) {
	var channelStatus []*ChannelStatus
	if err := db.Select(&channelStatus, "SELECT * from channelStatus ORDER by ts DESC"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return channelStatus, nil
}
