package twitch

import (
	"testing"
)

func TestCanParseRawMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :Thrashh5, FeelsWayTooAmazingMan kinda"

	message := parseMessage(testMessage)

	assertStringsEqual(t, "pajlada", message.Channel)
	assertStringsEqual(t, "redflamingo13", message.Username)
	if message.RawMessage.Type != PRIVMSG {
		t.Error("parsing message type failed")
	}
	assertStringsEqual(t, "PRIVMSG", message.RawMessage.RawType)
	assertStringsEqual(t, "subscriber/6,premium/1", message.RawMessage.Tags["badges"])
	assertStringsEqual(t, "#FF0000", message.RawMessage.Tags["color"])
	assertStringsEqual(t, "Redflamingo13", message.RawMessage.Tags["display-name"])
	assertStringsEqual(t, "", message.RawMessage.Tags["emotes"])
	assertStringsEqual(t, "2a31a9df-d6ff-4840-b211-a2547c7e656e", message.RawMessage.Tags["id"])
	assertStringsEqual(t, "11148817", message.RawMessage.Tags["room-id"])
	assertStringsEqual(t, "1490382457309", message.RawMessage.Tags["tmi-sent-ts"])
	assertStringsEqual(t, "78424343", message.RawMessage.Tags["user-id"])
	assertStringsEqual(t, "Thrashh5, FeelsWayTooAmazingMan kinda", message.RawMessage.Message)
}

func TestCanParsePRIVMSGMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :Thrashh5, FeelsWayTooAmazingMan kinda"

	message := parseMessage(testMessage)
	user, privateMessage := message.parsePRIVMSGMessage()

	assertStringsEqual(t, "pajlada", message.Channel)

	assertStringsEqual(t, "78424343", user.ID)
	assertStringsEqual(t, "redflamingo13", user.Name)
	assertStringsEqual(t, "Redflamingo13", user.DisplayName)
	assertStringsEqual(t, "#FF0000", user.Color)
	assertIntsEqual(t, 6, user.Badges["subscriber"])
	assertIntsEqual(t, 1, user.Badges["premium"])

	if privateMessage.Type != PRIVMSG {
		t.Error("parsing message type failed")
	}
	assertStringsEqual(t, "PRIVMSG", privateMessage.RawType)
	assertStringsEqual(t, "Thrashh5, FeelsWayTooAmazingMan kinda", privateMessage.Message)
	assertStringsEqual(t, "11148817", privateMessage.RoomID)
	assertStringsEqual(t, "2a31a9df-d6ff-4840-b211-a2547c7e656e", privateMessage.ID)
	assertFalse(t, privateMessage.Action, "parsing Action failed")
	assertIntsEqual(t, 0, len(privateMessage.Emotes))
	assertIntsEqual(t, 0, privateMessage.Bits)
}

func TestCanParseActionMessage(t *testing.T) {
	testMessage := "@badges=subscriber/6,premium/1;color=#FF0000;display-name=Redflamingo13;emotes=;id=2a31a9df-d6ff-4840-b211-a2547c7e656e;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1490382457309;turbo=0;user-id=78424343;user-type= :redflamingo13!redflamingo13@redflamingo13.tmi.twitch.tv PRIVMSG #pajlada :\u0001ACTION Thrashh5, FeelsWayTooAmazingMan kinda\u0001"

	message := parseMessage(testMessage)
	user, privateMessage := message.parsePRIVMSGMessage()

	assertStringsEqual(t, "pajlada", message.Channel)

	assertStringsEqual(t, "78424343", user.ID)
	assertStringsEqual(t, "redflamingo13", user.Name)
	assertStringsEqual(t, "Redflamingo13", user.DisplayName)
	assertStringsEqual(t, "#FF0000", user.Color)
	assertIntsEqual(t, 6, user.Badges["subscriber"])
	assertIntsEqual(t, 1, user.Badges["premium"])

	if privateMessage.Type != PRIVMSG {
		t.Error("parsing message type failed")
	}
	assertStringsEqual(t, "PRIVMSG", privateMessage.RawType)
	assertStringsEqual(t, "Thrashh5, FeelsWayTooAmazingMan kinda", privateMessage.Message)
	assertStringsEqual(t, "11148817", privateMessage.RoomID)
	assertStringsEqual(t, "2a31a9df-d6ff-4840-b211-a2547c7e656e", privateMessage.ID)
	assertTrue(t, privateMessage.Action, "parsing Action failed")
	assertIntsEqual(t, 0, len(privateMessage.Emotes))
	assertIntsEqual(t, 0, privateMessage.Bits)
}

func TestCanParseWHISPERMessage(t *testing.T) {
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv WHISPER gempir :i like memes"

	message := parseMessage(testMessage)
	user, whisperMessage := message.parseWHISPERMessage()

	assertStringsEqual(t, "32591953", user.ID)
	assertStringsEqual(t, "danielps1", user.Name)
	assertStringsEqual(t, "Danielps1", user.DisplayName)
	assertStringsEqual(t, "#00FF7F", user.Color)
	assertIntsEqual(t, 0, len(user.Badges))

	if whisperMessage.Type != WHISPER {
		t.Error("parsing message type failed")
	}
	assertStringsEqual(t, "WHISPER", whisperMessage.RawType)
	assertStringsEqual(t, "i like memes", whisperMessage.Message)
	assertIntsEqual(t, 0, len(whisperMessage.Emotes))
	assertFalse(t, whisperMessage.Action, "parsing action failed")
}

func TestCantParseNoTagsMessage(t *testing.T) {
	testMessage := "my test message"

	message := parseMessage(testMessage)

	assertStringsEqual(t, testMessage, message.RawMessage.Message)
}

func TestCantParseInvalidMessage(t *testing.T) {
	testMessage := "@my :test message"

	message := parseMessage(testMessage)

	assertStringsEqual(t, "", message.RawMessage.Message)
}

func TestCanParseCLEARCHATMessage(t *testing.T) {
	testMessage := `@room-id=408892348;tmi-sent-ts=1551290606462 :tmi.twitch.tv CLEARCHAT #clippyassistant`

	message := parseMessage(testMessage)
	clearchatMessage := message.parseCLEARCHATMessage()

	assertStringsEqual(t, message.Channel, "clippyassistant")

	if clearchatMessage.Type != CLEARCHAT {
		t.Error("parsing CLEARCHAT message failed")
	}
	assertStringsEqual(t, "CLEARCHAT", clearchatMessage.RawType)
	assertStringsEqual(t, "", clearchatMessage.Message)
	assertStringsEqual(t, "408892348", clearchatMessage.RoomID)
}

func TestCanParseBanMessage(t *testing.T) {
	testMessage := `@room-id=408892348;target-user-id=160087939;tmi-sent-ts=1551290659485 :tmi.twitch.tv CLEARCHAT #clippyassistant :nsfletcher`

	message := parseMessage(testMessage)
	clearchatMessage := message.parseCLEARCHATMessage()

	assertStringsEqual(t, message.Channel, "clippyassistant")

	if clearchatMessage.Type != CLEARCHAT {
		t.Error("parsing CLEARCHAT message failed")
	}
	assertStringsEqual(t, "CLEARCHAT", clearchatMessage.RawType)
	assertStringsEqual(t, "", clearchatMessage.Message)
	assertStringsEqual(t, "408892348", clearchatMessage.RoomID)
	assertIntsEqual(t, 0, clearchatMessage.BanDuration)
	assertStringsEqual(t, "160087939", clearchatMessage.TargetUserID)
	assertStringsEqual(t, "nsfletcher", clearchatMessage.TargetUsername)
}

func TestCanParseTimeoutMessage(t *testing.T) {
	testMessage := `@ban-duration=3;room-id=408892348;target-user-id=39489382;tmi-sent-ts=1551290562042 :tmi.twitch.tv CLEARCHAT #clippyassistant :taleof4gamers`

	message := parseMessage(testMessage)
	clearchatMessage := message.parseCLEARCHATMessage()

	assertStringsEqual(t, message.Channel, "clippyassistant")

	if clearchatMessage.Type != CLEARCHAT {
		t.Error("parsing CLEARCHAT message failed")
	}
	assertStringsEqual(t, "CLEARCHAT", clearchatMessage.RawType)
	assertStringsEqual(t, "", clearchatMessage.Message)
	assertStringsEqual(t, "408892348", clearchatMessage.RoomID)
	assertIntsEqual(t, 3, clearchatMessage.BanDuration)
	assertStringsEqual(t, "39489382", clearchatMessage.TargetUserID)
	assertStringsEqual(t, "taleof4gamers", clearchatMessage.TargetUsername)
}

func TestCanParseEmoteMessage(t *testing.T) {
	testMessage := "@badges=;color=#008000;display-name=Zugren;emotes=120232:0-6,13-19,26-32,39-45,52-58;id=51c290e9-1b50-497c-bb03-1667e1afe6e4;mod=0;room-id=11148817;sent-ts=1490382458685;subscriber=0;tmi-sent-ts=1490382456776;turbo=0;user-id=65897106;user-type= :zugren!zugren@zugren.tmi.twitch.tv PRIVMSG #pajlada :TriHard Clap TriHard Clap TriHard Clap TriHard Clap TriHard Clap"

	message := parseMessage(testMessage)
	_, privateMessage := message.parsePRIVMSGMessage()

	assertIntsEqual(t, 1, len(privateMessage.Emotes))
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

func TestCanParseJoinPart(t *testing.T) {
	testMessage := `:username123!username123@username123.tmi.twitch.tv JOIN #mychannel`

	channel, username := parseJoinPart(testMessage)

	assertStringsEqual(t, channel, "mychannel")
	assertStringsEqual(t, username, "username123")
}

func TestCanParseNames(t *testing.T) {
	testMessage := `:myusername123.tmi.twitch.tv 353 myusername123 = #mychannel :username1 username2 username3 username4`
	expectedUsers := []string{"username1", "username2", "username3", "username4"}

	channel, users := parseNames(testMessage)

	assertStringsEqual(t, channel, "mychannel")
	assertStringSlicesEqual(t, expectedUsers, users)
}
