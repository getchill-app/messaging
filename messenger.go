package messaging

import (
	"context"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/keys-pub/vault"
	"github.com/keys-pub/vault/syncer"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v4"
)

// Messenger ...
type Messenger struct {
	vault *vault.Vault
	init  bool
	smtx  sync.Mutex
}

func NewMessenger(vault *vault.Vault) *Messenger {
	return &Messenger{vault: vault}
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

func (m *Messenger) Vault() *vault.Vault {
	return m.vault
}

func (m *Messenger) check() error {
	if m.vault.DB() == nil {
		return vault.ErrLocked
	}
	if !m.init {
		if err := initTables(m.vault.DB()); err != nil {
			return err
		}
		m.init = true
	}
	return nil
}

func (m *Messenger) AddChannel(ctx context.Context, channel *keys.EdX25519Key, account *keys.EdX25519Key) (*api.Key, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	logger.Debugf("Add channel %s", channel.ID())
	return m.vault.Register(ctx, channel, account)
}

func (m *Messenger) AddKey(key *api.Key) error {
	if err := m.check(); err != nil {
		return err
	}
	logger.Debugf("Add key %s", key.ID)
	return m.vault.Keyring().Set(key)
}

func (m *Messenger) Key(kid keys.ID) (*api.Key, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return m.vault.Keyring().Get(kid)
}

func (m *Messenger) LeaveChannel(ctx context.Context, channel keys.ID) error {
	if err := m.check(); err != nil {
		return err
	}
	return updateChannelVisibility(m.vault.DB(), channel, VisibilityHidden)
}

func (m *Messenger) DeleteChannel(ctx context.Context, channel keys.ID) error {
	if err := m.check(); err != nil {
		return err
	}
	logger.Debugf("Leave channel %s", channel)

	if err := m.vault.Keyring().Remove(channel); err != nil {
		return err
	}

	err := syncer.Transact(m.vault.DB(), func(tx *sqlx.Tx) error {
		if err := deleteChannelTx(tx, channel); err != nil {
			return err
		}
		if err := deleteMessagesTx(tx, channel); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := m.vault.Keyring().Sync(ctx); err != nil {
		return err
	}

	return nil
}

// Add a message.
func (m *Messenger) AddMessage(msg *Message) error {
	if err := m.check(); err != nil {
		return err
	}

	channel, err := m.vault.Keyring().Key(msg.Channel)
	if err != nil {
		return err
	}
	senderKey, err := m.vault.Keyring().Key(msg.Sender)
	if err != nil {
		return err
	}

	cipher := NewSenderBox(senderKey.AsEdX25519())
	return syncer.Transact(m.vault.DB(), func(tx *sqlx.Tx) error {
		logger.Debugf("Saving message %s", msg.ID)
		b, err := msgpack.Marshal(msg)
		if err != nil {
			return err
		}
		// For pending message set remote index to max
		msg.RemoteIndex = 9223372036854775807
		if err := syncer.AddTx(tx, channel.AsEdX25519(), b, cipher); err != nil {
			return err
		}
		if err := addMessageTx(tx, msg); err != nil {
			return err
		}
		return nil
	})
}

// Send message.
func (m *Messenger) Send(ctx context.Context, msg *Message) error {
	logger.Debugf("Send message to %s", msg.Channel.ID())
	if err := m.AddMessage(msg); err != nil {
		return err
	}
	return m.SyncVault(ctx, msg.Channel)
}

func (m *Messenger) Messages(channel keys.ID) ([]*Message, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return getMessages(m.vault.DB(), channel)
}

func (m *Messenger) Channel(channel keys.ID) (*Channel, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return getChannel(m.vault.DB(), channel)
}

func (m *Messenger) Channels() ([]*Channel, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return getChannels(m.vault.DB())
}

// Sync all messages.
func (m *Messenger) Sync(ctx context.Context) error {
	m.smtx.Lock()
	defer m.smtx.Unlock()

	if err := m.check(); err != nil {
		return err
	}

	// Sync keyring
	logger.Infof("Sync keyring...")
	if err := m.vault.Keyring().Sync(ctx); err != nil {
		return err
	}

	// Get changes
	logger.Infof("Get changes...")
	chgs, err := m.vault.Changes(ctx)
	if err != nil {
		return err
	}
	logger.Infof("Found %d change(s)", len(chgs))

	// Sync each changed channel
	s := syncer.New(m.vault.DB(), m.vault.Client(), m.receive)
	for _, chg := range chgs {
		key, err := m.vault.Keyring().Key(chg.VID)
		if err != nil {
			return err
		}
		if err := s.Sync(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) SyncVault(ctx context.Context, vid keys.ID) error {
	m.smtx.Lock()
	defer m.smtx.Unlock()

	if err := m.check(); err != nil {
		return err
	}
	key, err := m.vault.Keyring().Key(vid)
	if err != nil {
		return err
	}

	logger.Debugf("Sync vault %s", key.ID)
	s := syncer.New(m.vault.DB(), m.vault.Client(), m.receive)
	if err := s.Sync(ctx, key); err != nil {
		return err
	}

	return nil
}

func (m *Messenger) receive(ctx *syncer.Context, events []*vault.Event) error {
	logger.Debugf("Key %s received %d event(s)", ctx.VID, len(events))
	key, err := m.vault.Keyring().Key(ctx.VID)
	if err != nil {
		return err
	}

	channel, err := getChannel(m.vault.DB(), ctx.VID)
	if err != nil {
		return err
	}
	if channel == nil {
		if err := insertChannelTx(ctx.Tx, ctx.VID); err != nil {
			return err
		}
		channel = &Channel{ID: ctx.VID}
	}

	for _, event := range events {
		b, pk, err := DecryptSenderBox(event.Data, key.AsEdX25519())
		if err != nil {
			return err
		}

		var msg Message
		if err := msgpack.Unmarshal(b, &msg); err != nil {
			return err
		}

		if !keys.X25519Match(msg.Sender, pk.ID()) {
			return errors.Errorf("message sender mismatch")
		}

		if err := addMessageTx(ctx.Tx, &msg); err != nil {
			return err
		}

		if err := updateChannelTx(ctx.Tx, channel.ID, msg.Text, msg.Timestamp, msg.RemoteTimestamp); err != nil {
			return err
		}

		if msg.Command != nil {
			if msg.Command.ChannelInfo != nil {
				if msg.Command.ChannelInfo.Name != "" {
					if err := updateChannelNameTx(ctx.Tx, channel.ID, msg.Command.ChannelInfo.Name); err != nil {
						return err
					}
				}
				if msg.Command.ChannelInfo.Description != "" {
					if err := updateChannelDescriptionTx(ctx.Tx, channel.ID, msg.Command.ChannelInfo.Description); err != nil {
						return err
					}
				}
			}
		}

	}

	return nil
}
