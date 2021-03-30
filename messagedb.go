package messaging

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"
)

func addMessageTx(tx *sqlx.Tx, msg *Message) error {
	if _, err := tx.Exec(`INSERT OR REPLACE INTO messages (id, channel, sender, ts, prev, text, cmd, ridx, rts) VALUES 
		($1, $2, $3, $4, $5, $6, $7, $8, $9);`,
		msg.ID,
		msg.Channel,
		msg.Sender,
		msg.Timestamp,
		msg.Prev,
		msg.Text,
		msg.Command,
		msg.RemoteIndex,
		msg.RemoteTimestamp); err != nil {
		return errors.Wrapf(err, "failed to insert message")
	}

	return nil
}

func getMessages(db *sqlx.DB, channel keys.ID) ([]*Message, error) {
	var out []*Message
	if err := db.Select(&out, "SELECT * FROM messages WHERE channel = $1 ORDER BY ridx, ts;", channel); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get messages")
	}
	return out, nil
}

func deleteMessagesTx(tx *sqlx.Tx, channel keys.ID) error {
	_, err := tx.Exec("DELETE from messages WHERE channel = ?", channel)
	return err
}
