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
		`CREATE TABLE IF NOT EXISTS channelStatus (
			channel TEXT PRIMARY KEY NOT NULL,
			name TEXT,			
			desc TEXT,
			snippet TEXT,			
			"index" INTEGER,
			readIndex INTEGER,
			ts INTEGER,
			rts INTEGER			
		);`,
		`CREATE INDEX IF NOT EXISTS index_channelStatus_ts
			ON channelStatus(ts desc);`,
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

func (m *Messenger) LeaveChannel(ctx context.Context, channel keys.ID) error {
	if err := m.check(); err != nil {
		return err
	}
	logger.Debugf("Leave channel %s", channel)

	if err := m.vault.Keyring().Remove(channel); err != nil {
		return err
	}

	err := syncer.Transact(m.vault.DB(), func(tx *sqlx.Tx) error {
		if err := deleteChannelStatusTx(tx, channel); err != nil {
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

// Save a message.
func (m *Messenger) Set(msg *Message) error {
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
	if err := m.Set(msg); err != nil {
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

func (m *Messenger) ChannelStatus(channel keys.ID) (*ChannelStatus, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return getChannelStatus(m.vault.DB(), channel)
}

func (m *Messenger) ChannelStatuses() ([]*ChannelStatus, error) {
	if err := m.check(); err != nil {
		return nil, err
	}
	return getChannelStatuses(m.vault.DB())
}

// Sync all messages.
// Returns error if sync is not enabled.
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

	status, err := getChannelStatus(m.vault.DB(), ctx.VID)
	if err != nil {
		return err
	}
	if status == nil {
		status = &ChannelStatus{Channel: ctx.VID}
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

		status.Update(&msg)
	}

	if err := updateChannelStatusTx(ctx.Tx, status); err != nil {
		return err
	}

	return nil
}

// func (m *Messenger) keysAsStatus() ([]*ChannelStatus, error) {
// 	if err := m.check(); err != nil {
// 		return nil, err
// 	}
// 	ks, err := m.vault.Keyring().Keys()
// 	if err != nil {
// 		return nil, err
// 	}
// 	out := []*ChannelStatus{}
// 	for _, k := range ks {
// 		if k.Token == "" {
// 			continue
// 		}
// 		st, err := getChannelStatus(m.vault.DB(), k.ID)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if st == nil {
// 			st = &ChannelStatus{Channel: k.ID, Name: "unknown"}
// 		}
// 		out = append(out, st)
// 	}
// 	return out, nil
// }
