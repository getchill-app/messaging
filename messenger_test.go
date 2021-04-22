package messaging_test

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/keys-pub/vault"
	"github.com/keys-pub/vault/testutil"
	"github.com/stretchr/testify/require"
)

func testMessenger(t *testing.T, env *testutil.Env, ck *api.Key) (*messaging.Messenger, func()) {
	vlt, closeFn := testutil.NewTestVaultWithSetup(t, env, "testpassword", ck)
	msgr := messaging.NewMessenger(vlt)
	return msgr, closeFn
}

func TestMessenger(t *testing.T) {
	// lg := vault.NewLogger(vault.DebugLevel)
	// messaging.SetLogger(lg)
	// vault.SetLogger(lg)
	// client.SetLogger(lg)

	var err error
	env := testutil.NewEnv(t, vault.ErrLevel)
	defer env.CloseFn()

	channel := keys.NewEdX25519KeyFromSeed(testutil.Seed(0xc0))
	t.Logf("Channel: %s", channel.ID())

	t.Logf("Alice")
	alice := keys.NewEdX25519KeyFromSeed(testutil.Seed(0x01))
	testutil.AccountCreate(t, env, alice, "alice@keys.pub")
	cka := testutil.RegisterClient(t, env, keys.NewEdX25519KeyFromSeed(testutil.Seed(0xa0)), alice)
	t.Logf("Alice client key: %s", cka.ID)
	aliceMsgr, aliceCloseFn := testMessenger(t, env, cka)
	defer aliceCloseFn()
	err = aliceMsgr.AddKey(api.NewKey(alice))
	require.NoError(t, err)

	_, err = aliceMsgr.AddChannel(context.TODO(), channel, alice)
	require.NoError(t, err)

	err = aliceMsgr.Send(context.TODO(), messaging.NewMessage(channel.ID(), alice.ID()).WithText("hi bob"))
	require.NoError(t, err)

	msgs1, err := aliceMsgr.Messages(channel.ID())
	require.NoError(t, err)
	require.Equal(t, 1, len(msgs1))

	t.Logf("Bob")
	bob := keys.NewEdX25519KeyFromSeed(testutil.Seed(0x02))
	testutil.AccountCreate(t, env, bob, "bob@keys.pub")
	ckb := testutil.RegisterClient(t, env, keys.NewEdX25519KeyFromSeed(testutil.Seed(0xa1)), bob)
	t.Logf("Bob client key: %s", ckb.ID)
	bobMsgr, bobCloseFn := testMessenger(t, env, ckb)
	defer bobCloseFn()
	err = bobMsgr.AddKey(api.NewKey(bob))
	require.NoError(t, err)

	_, err = bobMsgr.AddChannel(context.TODO(), channel, bob)
	require.NoError(t, err)

	err = bobMsgr.Sync(context.TODO())
	require.NoError(t, err)

	msgs2, err := bobMsgr.Messages(channel.ID())
	require.NoError(t, err)
	require.Equal(t, msgs1, msgs2)

	err = bobMsgr.Send(context.TODO(), messaging.NewMessageForChannelInfo(channel.ID(), bob.ID(), &messaging.ChannelInfo{Name: "testing"}))
	require.NoError(t, err)
	err = aliceMsgr.Send(context.TODO(), messaging.NewMessage(channel.ID(), alice.ID()).WithText("roses really smell like poopoo"))
	require.NoError(t, err)

	err = bobMsgr.Sync(context.TODO())
	require.NoError(t, err)

	ch, err := bobMsgr.Channel(channel.ID())
	require.NoError(t, err)
	require.Equal(t, channel.ID(), ch.ID)
	require.Equal(t, "testing", ch.Name)
	require.Equal(t, "roses really smell like poopoo", ch.Snippet)

	msgs3, err := bobMsgr.Messages(channel.ID())
	require.NoError(t, err)

	t.Logf("Alice #2")
	err = aliceMsgr.Sync(context.TODO())
	require.NoError(t, err)

	msgs4, err := aliceMsgr.Messages(channel.ID())
	require.NoError(t, err)
	require.Equal(t, msgs3, msgs4)
}

// To keep spew import
var _ = spew.Sdump("test")
