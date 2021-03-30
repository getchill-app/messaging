package messaging

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"
)

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

func deleteChannelStatusTx(tx *sqlx.Tx, channel keys.ID) error {
	_, err := tx.Exec("DELETE from channelStatus WHERE channel = ?", channel)
	return err
}
