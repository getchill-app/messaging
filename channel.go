package messaging

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"
)

type Channel struct {
	ID              keys.ID `json:"id" msgpack:"id" db:"id"`
	Name            string  `json:"name,omitempty" msgpack:"name,omitempty" db:"name"`
	Description     string  `json:"desc,omitempty" msgpack:"desc,omitempty" db:"desc"`
	Snippet         string  `json:"snippet,omitempty" msgpack:"snippet,omitempty" db:"snippet"`
	Index           int64   `json:"index,omitempty" msgpack:"index,omitempty" db:"index"`
	Timestamp       int64   `json:"ts,omitempty" msgpack:"ts,omitempty" db:"ts"`
	RemoteTimestamp int64   `json:"rts,omitempty" msgpack:"rts,omitempty" db:"rts"`
	ReadIndex       int64   `json:"readIndex,omitempty" msgpack:"readIndex,omitempty" db:"readIndex"`
	Visibility      int     `json:"visibility,omitempty" msgpack:"visibility,omitempty" db:"visibility"`
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

func updateChannelTx(tx *sqlx.Tx, id keys.ID, snippet string, ts int64, rts int64) error {
	if _, err := tx.Exec(`UPDATE channels SET snippet=?, ts=?, rts=? WHERE id=?`, snippet, ts, rts, id); err != nil {
		return errors.Wrapf(err, "failed to update channel")
	}
	return nil
}

func updateChannelNameTx(tx *sqlx.Tx, id keys.ID, name string) error {
	if _, err := tx.Exec(`UPDATE channels SET name=? WHERE id=?`, name, id); err != nil {
		return errors.Wrapf(err, "failed to update channel")
	}
	return nil
}

func updateChannelDescriptionTx(tx *sqlx.Tx, id keys.ID, desc string) error {
	if _, err := tx.Exec(`UPDATE channels SET desc=? WHERE id=?`, desc, id); err != nil {
		return errors.Wrapf(err, "failed to update channel")
	}
	return nil
}

func updateChannelVisibility(db *sqlx.DB, id keys.ID, visibility int) error {
	if _, err := db.Exec(`UPDATE channels SET visibility=? WHERE id=?`, visibility, id); err != nil {
		return errors.Wrapf(err, "failed to update channel")
	}
	return nil
}

func getChannel(db *sqlx.DB, id keys.ID) (*Channel, error) {
	var channel Channel
	if err := db.Get(&channel, "SELECT * from channels WHERE id=?", id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

func getChannels(db *sqlx.DB) ([]*Channel, error) {
	var channels []*Channel
	if err := db.Select(&channels, "SELECT * from channels ORDER by ts DESC"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return channels, nil
}

func deleteChannelTx(tx *sqlx.Tx, id keys.ID) error {
	_, err := tx.Exec("DELETE from channels WHERE id = ?", id)
	return err
}
