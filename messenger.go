package messaging

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/getchill-app/http/api"
	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/pkg/errors"

	// For sqlite3 (sqlcipher driver)
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// Messenger ...
type Messenger struct {
	db *sqlx.DB
}

func NewMessenger(path string, mk *[32]byte) (*Messenger, error) {
	db, err := openDB(path, mk)
	if err != nil {
		return nil, err
	}
	if err := initTables(db); err != nil {
		return nil, err
	}
	return &Messenger{db: db}, nil
}

func openDB(path string, mk *[32]byte) (*sqlx.DB, error) {
	keyString := hex.EncodeToString(mk[:])
	pragma := fmt.Sprintf("?_pragma_key=x'%s'&_pragma_cipher_page_size=4096", keyString)

	db, err := sqlx.Open("sqlite3", path+pragma)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db")
	}

	return db, nil
}

func initTables(db *sqlx.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY NOT NULL,
			channel TEXT NOT NULL,			
			text TEXT,
			sender TEXT,
			prev TEXT,
			cmd BLOB,
			ts INTEGER,
			rts INTEGER,
			ridx INTEGER
		);`,
		`CREATE INDEX IF NOT EXISTS index_messages_channel_ridx
			ON messages(channel, ridx);`,
		`CREATE TABLE IF NOT EXISTS channels (
			id TEXT PRIMARY KEY NOT NULL,
			name TEXT DEFAULT '',		
			desc TEXT DEFAULT '',
			snippet TEXT DEFAULT '',			
			"index" INTEGER DEFAULT 0,
			readIndex INTEGER DEFAULT 0,
			ts INTEGER DEFAULT 0,
			rts INTEGER DEFAULT 0,
			visibility INTEGER DEFAULT 0		
		);`,
		`CREATE INDEX IF NOT EXISTS index_channels_ts
			ON channels(ts desc);`,
		`CREATE TABLE IF NOT EXISTS users (
			kid TEXT PRIMARY KEY NOT NULL,
			username TEXT				
		);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) AddChannel(cid keys.ID) error {
	logger.Debugf("Add channel %s", cid)
	return Transact(m.db, func(tx *sqlx.Tx) error {
		return insertChannelTx(tx, cid)
	})
}

func (m *Messenger) HideChannel(ctx context.Context, channel keys.ID) error {
	return updateChannelVisibility(m.db, channel, VisibilityHidden)
}

func (m *Messenger) DeleteChannel(ctx context.Context, kid keys.ID) error {
	logger.Debugf("Delete channel %s", kid)
	err := Transact(m.db, func(tx *sqlx.Tx) error {
		if err := deleteChannelTx(tx, kid); err != nil {
			return err
		}
		if err := deleteMessagesTx(tx, kid); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Add a pending message.
func (m *Messenger) AddPending(msg *api.Message) error {
	return Transact(m.db, func(tx *sqlx.Tx) error {
		logger.Debugf("Add pending message %s", msg.ID)
		// For pending message set remote index to max
		msg.RemoteIndex = 9223372036854775807
		if err := addMessageTx(tx, msg); err != nil {
			return err
		}
		return nil
	})
}

func (m *Messenger) Messages(channel keys.ID) ([]*api.Message, error) {
	return getMessages(m.db, channel)
}

func (m *Messenger) Channel(channel keys.ID) (*Channel, error) {
	return getChannel(m.db, channel)
}

func (m *Messenger) Channels() ([]*Channel, error) {
	return getChannels(m.db)
}

func (m *Messenger) AddMessages(cid keys.ID, messages []*api.Message) error {
	channel, err := getChannel(m.db, cid)
	if err != nil {
		return err
	}
	if channel == nil {
		return errors.Errorf("no channel")
	}

	return Transact(m.db, func(tx *sqlx.Tx) error {
		for _, msg := range messages {
			if err := addMessageTx(tx, msg); err != nil {
				return err
			}

			if err := updateChannelTx(tx, channel.ID, msg.Text, msg.Timestamp, msg.RemoteTimestamp); err != nil {
				return err
			}

			if msg.Command != nil {
				if msg.Command.ChannelInfo != nil {
					if msg.Command.ChannelInfo.Name != "" {
						if err := updateChannelNameTx(tx, channel.ID, msg.Command.ChannelInfo.Name); err != nil {
							return err
						}
					}
					if msg.Command.ChannelInfo.Description != "" {
						if err := updateChannelDescriptionTx(tx, channel.ID, msg.Command.ChannelInfo.Description); err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	})

}
