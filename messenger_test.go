package messaging_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/getchill-app/http/api"
	"github.com/getchill-app/http/client/testutil"
	"github.com/getchill-app/http/server"
	"github.com/getchill-app/messaging"
	"github.com/keys-pub/keys"
	"github.com/stretchr/testify/require"
)

func TestMessenger(t *testing.T) {
	env, closeFn := testutil.NewEnv(t, server.NoLevel)
	defer closeFn()
	emailer := testutil.NewTestEmailer()
	env.SetEmailer(emailer)
	ctx := context.TODO()
	var err error

	aliceClient := testutil.NewTestClient(t, env)
	alice := keys.NewEdX25519KeyFromSeed(testutil.Seed(0x01))

	testutil.TestAccount(t, aliceClient, emailer, alice, "alice@keys.pub", "alice")

	path := testPath()
	mk := keys.Rand32()
	messenger, err := messaging.NewMessenger(path, mk)
	require.NoError(t, err)

	channel := keys.NewEdX25519KeyFromSeed(testutil.Seed(0xc0))

	err = messenger.AddChannel(channel.ID())
	require.NoError(t, err)

	_, err = aliceClient.ChannelCreate(ctx, channel, alice)
	require.NoError(t, err)

	msg := api.NewMessage(channel.ID(), alice.ID()).WithText("hi bob")
	err = messenger.AddPending(msg)
	require.NoError(t, err)
	err = aliceClient.SendMessage(ctx, msg, channel, alice)
	require.NoError(t, err)

	ch, err := messenger.Channel(channel.ID())
	require.NoError(t, err)
	require.NotNil(t, ch)
	msgs, err := aliceClient.Messages(ctx, channel, ch.Index)
	require.NoError(t, err)

	err = messenger.AddMessages(channel.ID(), msgs.Messages)
	require.NoError(t, err)

	out, err := messenger.Messages(channel.ID())
	require.NoError(t, err)
	msg.RemoteIndex = out[0].RemoteIndex
	msg.RemoteTimestamp = out[0].RemoteTimestamp

	require.Equal(t, out, []*api.Message{msg})
}

// To keep spew import
var _ = spew.Sdump("test")

func testPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s.db", keys.RandFileName()))
}
