package messaging_test

import (
	"context"
	"testing"

	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	"github.com/keys-pub/keys/api"
	"github.com/keys-pub/vault/testutil"
	"github.com/stretchr/testify/require"
)

func testMessenger(t *testing.T, env *testutil.Env, ck *keys.EdX25519Key) (*messaging.Messenger, func()) {
	vlt, closeFn := testutil.NewTestVaultWithSetup(t, env, "testpassword", ck)
	msgr := messaging.NewMessenger(vlt)

	return msgr, closeFn
}

func TestMessenger(t *testing.T) {
	// lg := vault.NewLogger(vault.DebugLevel)
	// messaging.SetLogger(lg)
	// vault.SetLogger(lg)

	var err error
	env := testutil.NewEnv(t, nil) // vault.NewLogger(vault.DebugLevel))
	defer env.CloseFn()

	channel := keys.NewEdX25519KeyFromSeed(testutil.Seed(0xa0))
	t.Logf("Channel: %s", channel.ID())

	t.Logf("Alice")
	cka := keys.NewEdX25519KeyFromSeed(testutil.Seed(0x60))
	alice := keys.NewEdX25519KeyFromSeed(testutil.Seed(0x01))
	aliceMsgr, aliceCloseFn := testMessenger(t, env, cka)
	defer aliceCloseFn()
	err = aliceMsgr.AddKey(api.NewKey(alice))
	require.NoError(t, err)

	_, err = aliceMsgr.AddChannel(context.TODO(), channel)
	require.NoError(t, err)

	err = aliceMsgr.Send(context.TODO(), messaging.NewMessage(channel.ID(), alice.ID()).WithText("hi bob"))
	require.NoError(t, err)

	msgs1, err := aliceMsgr.Messages(channel.ID())
	require.NoError(t, err)
	require.Equal(t, 1, len(msgs1))

	t.Logf("Bob")
	ckb := keys.NewEdX25519KeyFromSeed(testutil.Seed(0x61))
	bob := keys.NewEdX25519KeyFromSeed(testutil.Seed(0x02))
	bobMsgr, bobCloseFn := testMessenger(t, env, ckb)
	defer bobCloseFn()
	err = bobMsgr.AddKey(api.NewKey(bob))
	require.NoError(t, err)

	_, err = bobMsgr.AddChannel(context.TODO(), channel)
	require.NoError(t, err)

	err = bobMsgr.Sync(context.TODO())
	require.NoError(t, err)

	msgs2, err := bobMsgr.Messages(channel.ID())
	require.NoError(t, err)
	require.Equal(t, msgs1, msgs2)

	err = bobMsgr.Send(context.TODO(), messaging.NewMessage(channel.ID(), bob.ID()).WithText("what's the password?"))
	require.NoError(t, err)
	err = aliceMsgr.Send(context.TODO(), messaging.NewMessage(channel.ID(), alice.ID()).WithText("roses really smell like poopoo"))
	require.NoError(t, err)

	err = bobMsgr.Sync(context.TODO())
	require.NoError(t, err)

	status, err := bobMsgr.ChannelStatus((channel.ID()))
	require.NoError(t, err)
	require.Equal(t, channel.ID(), status.Channel)
	require.Equal(t, "roses really smell like poopoo", status.Snippet)

	msgs3, err := bobMsgr.Messages(channel.ID())
	require.NoError(t, err)

	t.Logf("Alice #2")
	err = aliceMsgr.Sync(context.TODO())
	require.NoError(t, err)

	msgs4, err := aliceMsgr.Messages(channel.ID())
	require.NoError(t, err)
	require.Equal(t, msgs3, msgs4)

	t.Logf("Bob leave")
	err = bobMsgr.LeaveChannel(context.TODO(), channel.ID())
	require.NoError(t, err)
	check, err := bobMsgr.Vault().Keyring().Get(channel.ID())
	require.NoError(t, err)
	require.Nil(t, check)

}
