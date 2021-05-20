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

	testutil.TestAccount(t, aliceClient, emailer, alice, "alice@keys.pub", "", "alice")

	path := testPath()
	mk := keys.Rand32()
	messenger, err := messaging.NewMessenger(path, mk)
	require.NoError(t, err)

	channelKey := keys.NewEdX25519KeyFromSeed(testutil.Seed(0xc0))

	info := &api.ChannelInfo{Name: "testing"}
	_, err = aliceClient.ChannelCreateWithUsers(ctx, channelKey, info, []keys.ID{alice.ID()}, alice)
	require.NoError(t, err)

	msg := api.NewMessage(channelKey.ID(), alice.ID()).WithText("Sartorial taxidermy irony ramps mixtape YOLO. Vape hella 90's VHS jianbing mumblecore, roof party ugh kogi cray occupy kombucha blue bottle.")
	err = messenger.AddPending(msg)
	require.NoError(t, err)
	err = aliceClient.SendMessage(ctx, msg, channelKey, alice)
	require.NoError(t, err)
	msg2 := api.NewMessage(channelKey.ID(), alice.ID()).WithText("8-bit shabby chic ugh hella fanny pack pour-over PBR&B ennui")
	err = messenger.AddPending(msg2)
	require.NoError(t, err)
	err = aliceClient.SendMessage(ctx, msg2, channelKey, alice)
	require.NoError(t, err)

	msgs, err := aliceClient.Messages(ctx, channelKey, 0)
	require.NoError(t, err)

	// Add channel
	ch, err := aliceClient.Channel(ctx, channelKey, alice)
	require.NoError(t, err)
	channel, err := messaging.NewChannelFromAPI(ch, channelKey)
	require.NoError(t, err)
	err = messenger.AddChannel(channel)
	require.NoError(t, err)

	err = messenger.AddMessages(channelKey.ID(), msgs.Messages)
	require.NoError(t, err)

	channel, err = messenger.Channel(channelKey.ID())
	require.NoError(t, err)
	require.Equal(t, int64(2), channel.MessageIndex)

	out, err := messenger.Messages(channelKey.ID())
	require.NoError(t, err)
	require.Equal(t, 2, len(out))
	msg.RemoteIndex = out[0].RemoteIndex
	msg.RemoteTimestamp = out[0].RemoteTimestamp
	msg2.RemoteIndex = out[1].RemoteIndex
	msg2.RemoteTimestamp = out[1].RemoteTimestamp
	require.Equal(t, out, []*api.Message{msg, msg2})

	results, err := messenger.Search("mumblecore")
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	require.Equal(t, msg.ID, results[0].ID)
}

// To keep spew import
var _ = spew.Sdump("test")

func testPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s.db", keys.RandFileName()))
}
