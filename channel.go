package messaging

import (
	"database/sql"
	"encoding/json"

	"github.com/getchill-app/http/api"
	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"
)

type channelRow struct {
	ID      keys.ID `db:"id"`
	Channel []byte  `db:"channel"`
	Last    []byte  `db:"last"`
}

type Channel struct {
	ID              keys.ID `json:"id" msgpack:"id"`
	Name            string  `json:"name,omitempty" msgpack:"name,omitempty"`
	Description     string  `json:"desc,omitempty" msgpack:"desc,omitempty"`
	Topic           string  `json:"topic,omitempty" msgpack:"topic,omitempty"`
	Snippet         string  `json:"snippet,omitempty" msgpack:"snippet,omitempty"`
	Index           int64   `json:"index,omitempty" msgpack:"index,omitempty"`
	Timestamp       int64   `json:"ts,omitempty" msgpack:"ts,omitempty"`
	RemoteTimestamp int64   `json:"rts,omitempty" msgpack:"rts,omitempty"`
	ReadIndex       int64   `json:"readIndex,omitempty" msgpack:"readIndex,omitempty"`
}

const (
	VisibilityHidden int = 1
)

func insertChannelTx(tx *sqlx.Tx, id keys.ID) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO channels (id) VALUES ($1);`, id); err != nil {
		return errors.Wrapf(err, "failed to insert channel")
	}
	return nil
}

func updateChannelTx(tx *sqlx.Tx, channel *Channel) error {
	b, _ := json.Marshal(channel)
	if _, err := tx.Exec(`UPDATE channels SET channel=? WHERE id=?`, b, channel.ID); err != nil {
		return errors.Wrapf(err, "failed to update channel")
	}
	return nil
}

func updateLastMessageTx(tx *sqlx.Tx, id keys.ID, last *api.Message) error {
	b, _ := json.Marshal(last)
	if _, err := tx.Exec(`UPDATE channels SET last=? WHERE id=?`, b, id); err != nil {
		return errors.Wrapf(err, "failed to update last message")
	}
	return nil
}

func getChannel(db *sqlx.DB, id keys.ID) (*Channel, error) {
	var row channelRow
	if err := db.Get(&row, "SELECT * from channels WHERE id=?", id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rowToChannel(&row)
}

func rowToChannel(row *channelRow) (*Channel, error) {
	channel := Channel{
		ID: row.ID,
	}
	if row.Channel != nil {
		if err := json.Unmarshal(row.Channel, &channel); err != nil {
			return nil, err
		}
	}
	if row.Last != nil {
		var last api.Message
		if err := json.Unmarshal(row.Last, &last); err != nil {
			return nil, err
		}
		channel.Snippet = last.Text
		channel.Timestamp = last.Timestamp
		channel.RemoteTimestamp = last.RemoteTimestamp
		channel.Index = last.RemoteIndex
	}

	return &channel, nil
}

func getChannels(db *sqlx.DB) ([]*Channel, error) {
	var rows []*channelRow
	if err := db.Select(&rows, "SELECT * from channels"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	channels := []*Channel{}
	for _, row := range rows {
		channel, err := rowToChannel(row)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

func deleteChannelTx(tx *sqlx.Tx, id keys.ID) error {
	_, err := tx.Exec("DELETE from channels WHERE id = ?", id)
	return err
}
