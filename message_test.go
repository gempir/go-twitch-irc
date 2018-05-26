package twitch

import (
	"testing"
)

func TestCanParseMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :Thrashh5, FeelsWayTooAmazingMan kinda"
	message := parseMessage(testMessage)

	assertStringsEqual(t, "pajlada", message.Channel)
	assertInts64Equal(t, 78424343, message.UserID)
	assertIntsEqual(t, 6, message.Badges["subscriber"])
	assertStringsEqual(t, "#FF0000", message.Color)
	assertStringsEqual(t, "Redflamingo13", message.DisplayName)
	assertIntsEqual(t, 0, len(message.Emotes))
	assertStringsEqual(t, "0", message.Tags["mod"])
	assertStringsEqual(t, "Thrashh5, FeelsWayTooAmazingMan kinda", message.Text)
	if message.Type != PRIVMSG {
		t.Error("parsing message type failed")
	}
	assertStringsEqual(t, "redflamingo13", message.Username)
	assertStringsEqual(t, "", message.UserType)
	assertFalse(t, message.Action, "parsing action failed")
}

func TestCanParseActionMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :\u0001ACTION Thrashh5, FeelsWayTooAmazingMan kinda\u0001"
	message := parseMessage(testMessage)

	assertStringsEqual(t, "pajlada", message.Channel)
	assertIntsEqual(t, 6, message.Badges["subscriber"])
	assertStringsEqual(t, "#FF0000", message.Color)
	assertStringsEqual(t, "Redflamingo13", message.DisplayName)
	assertIntsEqual(t, 0, len(message.Emotes))
	assertStringsEqual(t, "0", message.Tags["mod"])
	assertStringsEqual(t, "Thrashh5, FeelsWayTooAmazingMan kinda", message.Text)
	if message.Type != PRIVMSG {
		t.Error("parsing message type failed")
	}
	assertStringsEqual(t, "redflamingo13", message.Username)
	assertStringsEqual(t, "", message.UserType)
	assertTrue(t, message.Action, "parsing action failed")
}

func TestCanParseWhisper(t *testing.T) {
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv WHISPER gempir :i like memes"
	message := parseMessage(testMessage)

	assertIntsEqual(t, 0, message.Badges["subscriber"])
	assertStringsEqual(t, "#00FF7F", message.Color)
	assertStringsEqual(t, "Danielps1", message.DisplayName)
	assertIntsEqual(t, 0, len(message.Emotes))
	assertStringsEqual(t, "", message.Tags["mod"])
	assertStringsEqual(t, "i like memes", message.Text)
	if message.Type != WHISPER {
		t.Error("parsing message type failed")
	}
	assertStringsEqual(t, "danielps1", message.Username)
	assertFalse(t, message.Action, "parsing action failed")
}

func TestCantParseNoTagsMessage(t *testing.T) {
	testMessage := "my test message"

	message := parseMessage(testMessage)

	assertStringsEqual(t, testMessage, message.Text)
}

func TestCantParseInvalidMessage(t *testing.T) {
	testMessage := "@my :test message"

	message := parseMessage(testMessage)

	assertStringsEqual(t, "", message.Text)
}

func TestCanParseClearChatMessage(t *testing.T) {
	testMessage := `@ban-duration=1;ban-reason=testing\sxd;room-id=11148817;target-user-id=40910607 :tmi.twitch.tv CLEARCHAT #pajlada :ampzyh`

	message := parseMessage(testMessage)

	if message.Type != CLEARCHAT {
		t.Error("parsing CLEARCHAT message failed")
	}

	assertStringsEqual(t, message.Channel, "pajlada")
}

func TestCanParseClearChatMessage2(t *testing.T) {
	testMessage := `@room-id=11148817;tmi-sent-ts=1527342985836 :tmi.twitch.tv CLEARCHAT #pajlada`

	message := parseMessage(testMessage)

	if message.Type != CLEARCHAT {
		t.Error("parsing CLEARCHAT message failed")
	}

	assertStringsEqual(t, message.Channel, "pajlada")
}

func TestCanParseEmoteMessage(t *testing.T) {
	testMessage := "@badges=;color=#008000;display-name=Zugren;emotes=120232:0-6,13-19,26-32,39-45,52-58;id=51c290e9-1b50-497c-bb03-1667e1afe6e4;mod=0;room-id=11148817;sent-ts=1490382458685;subscriber=0;tmi-sent-ts=1490382456776;turbo=0;user-id=65897106;user-type= :zugren!zugren@zugren.tmi.twitch.tv PRIVMSG #pajlada :TriHard Clap TriHard Clap TriHard Clap TriHard Clap TriHard Clap"

	message := parseMessage(testMessage)

	assertIntsEqual(t, 1, len(message.Emotes))
}

func TestCanParseUserNoticeMessage(t *testing.T) {
	testMessage := `@badges=moderator/1,subscriber/24,premium/1;color=#33FFFF;display-name=Baxx;emotes=;id=4d737a10-03ff-48a7-aca1-a5624ebac91d;login=baxx;mod=1;msg-id=subgift;msg-param-months=7;msg-param-recipient-display-name=Nclnat;msg-param-recipient-id=84027795;msg-param-recipient-user-name=nclnat;msg-param-sender-count=7;msg-param-sub-plan-name=look\sat\sthose\sshitty\semotes,\srip\s$5\sLUL;msg-param-sub-plan=1000;room-id=11148817;subscriber=1;system-msg=Baxx\sgifted\sa\sTier\s1\ssub\sto\sNclnat!\sThey\shave\sgiven\s7\sGift\sSubs\sin\sthe\schannel!;tmi-sent-ts=1527341500077;turbo=0;user-id=59504812;user-type=mod :tmi.twitch.tv USERNOTICE #pajlada`
	message := parseMessage(testMessage)

	if message.Type != USERNOTICE {
		t.Error("parsing USERNOTICE message failed")
	}

	assertStringsEqual(t, message.Tags["msg-id"], "subgift")
	assertStringsEqual(t, message.Channel, "pajlada")
}

func TestCanParseUserNoticeRaidMessage(t *testing.T) {
	testMessage := `@badges=turbo/1;color=#9ACD32;display-name=TestChannel;emotes=;id=3d830f12-795c-447d-af3c-ea05e40fbddb;login=testchannel;mod=0;msg-id=raid;msg-param-displayName=TestChannel;msg-param-login=testchannel;msg-param-viewerCount=15;room-id=56379257;subscriber=0;system-msg=15\sraiders\sfrom\sTestChannel\shave\sjoined\n!;tmi-sent-ts=1507246572675;tmi-sent-ts=1507246572675;turbo=1;user-id=123456;user-type= :tmi.twitch.tv USERNOTICE #othertestchannel`
	message := parseMessage(testMessage)

	if message.Type != USERNOTICE {
		t.Error("parsing USERNOTICE message failed")
	}

	assertStringsEqual(t, message.Tags["msg-id"], "raid")
	assertStringsEqual(t, message.Channel, "othertestchannel")
}

func TestCanParseRoomstateMessage(t *testing.T) {
	testMessage := `@broadcaster-lang=<broadcaster-lang>;r9k=<r9k>;slow=<slow>;subs-only=<subs-only> :tmi.twitch.tv ROOMSTATE #nothing`

	message := parseMessage(testMessage)

	if message.Type != ROOMSTATE {
		t.Error("parsing ROOMSTATE message failed")
	}

	assertStringsEqual(t, message.Channel, "nothing")
}
