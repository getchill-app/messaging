package messaging

import (
	"database/sql"
	"encoding/json"

	"github.com/getchill-app/http/api"
	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"
)

func addMessageTx(tx *sqlx.Tx, msg *api.Message) error {
	b, _ := json.Marshal(msg)
	if _, err := tx.Exec(`INSERT OR REPLACE INTO messages (id, channel, sender, ts, rts, ridx, message) VALUES 
		($1, $2, $3, $4, $5, $6, $7);`,
		msg.ID,
		msg.Channel,
		msg.Sender,
		msg.Timestamp,
		msg.RemoteTimestamp,
		msg.RemoteIndex,
		b); err != nil {
		return errors.Wrapf(err, "failed to insert message")
	}

	if _, err := tx.Exec(`INSERT OR REPLACE INTO messages_fts (id, text) VALUES ($1, $2)`, msg.ID, msg.Text); err != nil {
		return errors.Wrapf(err, "failed to index message")
	}

	return nil
}

func getMessages(db *sqlx.DB, channel keys.ID) ([]*api.Message, error) {
	var rows []struct {
		ID              keys.ID `db:"id"`
		Channel         keys.ID `db:"channel"`
		Sender          keys.ID `db:"sender"`
		Timestamp       int64   `db:"ts"`
		RemoteTimestamp int64   `db:"rts"`
		RemoteIndex     int64   `db:"ridx"`
		Message         []byte  `db:"message"`
	}
	if err := db.Select(&rows, "SELECT * FROM messages WHERE channel = $1 ORDER BY ridx, ts;", channel); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get messages")
	}
	out := make([]*api.Message, 0, len(rows))
	for _, row := range rows {
		var msg api.Message
		if err := json.Unmarshal(row.Message, &msg); err != nil {
			return nil, err
		}
		out = append(out, &msg)
	}
	return out, nil
}

func deleteMessagesTx(tx *sqlx.Tx, channel keys.ID) error {
	_, err := tx.Exec("DELETE from messages WHERE channel = ?", channel)
	return err
}

type SearchResult struct {
	ID string
}

func searchMessages(db *sqlx.DB, text string) ([]*SearchResult, error) {
	var rows []struct {
		ID   string `db:"id"`
		Text string `db:"text"`
	}
	if err := db.Select(&rows, "SELECT DISTINCT * FROM messages_fts WHERE messages_fts MATCH $1", text); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get messages")
	}

	out := make([]*SearchResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, &SearchResult{ID: row.ID})
	}
	return out, nil
}
