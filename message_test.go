package twitch

import (
	"testing"
)

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

	expectedBadges := map[string]int{
		"subscriber": 6,
		"premium":    1,
	}
	assertStringIntMapsEqual(t, expectedBadges, user.Badges)

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
	_, privateMessage := message.parsePRIVMSGMessage()

	assertTrue(t, privateMessage.Action, "parsing Action failed")
}

func TestCanParseEmoteMessage(t *testing.T) {
	testMessage := "@badges=;color=#008000;display-name=Zugren;emotes=120232:0-6,13-19,26-32,39-45,52-58;id=51c290e9-1b50-497c-bb03-1667e1afe6e4;mod=0;room-id=11148817;sent-ts=1490382458685;subscriber=0;tmi-sent-ts=1490382456776;turbo=0;user-id=65897106;user-type= :zugren!zugren@zugren.tmi.twitch.tv PRIVMSG #pajlada :TriHard Clap TriHard Clap TriHard Clap TriHard Clap TriHard Clap"

	message := parseMessage(testMessage)
	_, privateMessage := message.parsePRIVMSGMessage()

	assertIntsEqual(t, 1, len(privateMessage.Emotes))
}

func TestCanParseWHISPERMessage(t *testing.T) {
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv WHISPER gempir :i like memes"

	message := parseMessage(testMessage)
	user, whisperMessage := message.parseWHISPERMessage()

	assertStringsEqual(t, "32591953", user.ID)
	assertStringsEqual(t, "danielps1", user.Name)
	assertStringsEqual(t, "Danielps1", user.DisplayName)
	assertStringsEqual(t, "#00FF7F", user.Color)

	expectedBadges := map[string]int{}
	assertStringIntMapsEqual(t, expectedBadges, user.Badges)

	if whisperMessage.Type != WHISPER {
		t.Error("parsing message type failed")
	}
	assertStringsEqual(t, "WHISPER", whisperMessage.RawType)
	assertStringsEqual(t, "i like memes", whisperMessage.Message)
	assertIntsEqual(t, 0, len(whisperMessage.Emotes))
	assertFalse(t, whisperMessage.Action, "parsing action failed")
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

func TestCanParseROOMSTATEMessage(t *testing.T) {
	testMessage := `@broadcaster-lang=en;emote-only=0;followers-only=10;r9k=0;rituals=0;room-id=408892348;slow=0;subs-only=0 :tmi.twitch.tv ROOMSTATE #clippyassistant`

	message := parseMessage(testMessage)
	roomstateMessage := message.parseROOMSTATEMessage()

	if roomstateMessage.Type != ROOMSTATE {
		t.Error("parsing ROOMSTATE message failed")
	}
	assertStringsEqual(t, "ROOMSTATE", roomstateMessage.RawType)
	assertStringsEqual(t, "", roomstateMessage.Message)
	assertStringsEqual(t, "clippyassistant", roomstateMessage.Channel)
	assertStringsEqual(t, "408892348", roomstateMessage.RoomID)
	assertStringsEqual(t, "en", roomstateMessage.Language)

	expectedState := map[string]int{
		"emote-only":     0,
		"followers-only": 10,
		"r9k":            0,
		"rituals":        0,
		"slow":           0,
		"subs-only":      0,
	}
	assertStringIntMapsEqual(t, expectedState, roomstateMessage.State)
}

func TestCanParseROOMSTATEChangeMessage(t *testing.T) {
	testMessage := `@followers-only=-1;room-id=408892348 :tmi.twitch.tv ROOMSTATE #clippyassistant`

	message := parseMessage(testMessage)
	roomstateMessage := message.parseROOMSTATEMessage()

	assertStringsEqual(t, "", roomstateMessage.Language)

	expectedState := map[string]int{
		"followers-only": -1,
	}
	assertStringIntMapsEqual(t, expectedState, roomstateMessage.State)

}

func TestCanParseUSERNOTICESubMessage(t *testing.T) {
	testMessage := `@badges=staff/1,broadcaster/1,turbo/1;color=#008000;display-name=ronni;emotes=;id=db25007f-7a18-43eb-9379-80131e44d633;login=ronni;mod=0;msg-id=resub;msg-param-cumulative-months=6;msg-param-streak-months=2;msg-param-should-share-streak=1;msg-param-sub-plan=Prime;msg-param-sub-plan-name=Prime;room-id=1337;subscriber=1;system-msg=ronni\shas\ssubscribed\sfor\s6\smonths!;tmi-sent-ts=1507246572675;turbo=1;user-id=1337;user-type=staff :tmi.twitch.tv USERNOTICE #dallas :Great stream -- keep it up!`

	message := parseMessage(testMessage)
	user, usernoticeMessage := message.parseUSERNOTICEMessage()

	assertStringsEqual(t, "1337", user.ID)
	assertStringsEqual(t, "ronni", user.Name)
	assertStringsEqual(t, "ronni", user.DisplayName)
	assertStringsEqual(t, "#008000", user.Color)

	expectedBadges := map[string]int{
		"staff":       1,
		"broadcaster": 1,
		"turbo":       1,
	}
	assertStringIntMapsEqual(t, expectedBadges, user.Badges)

	if usernoticeMessage.Type != USERNOTICE {
		t.Error("parsing USERNOTICE message failed")
	}
	assertStringsEqual(t, "USERNOTICE", usernoticeMessage.RawType)
	assertStringsEqual(t, "Great stream -- keep it up!", usernoticeMessage.Message)
	assertStringsEqual(t, "dallas", usernoticeMessage.Channel)
	assertStringsEqual(t, "1337", usernoticeMessage.RoomID)
	assertStringsEqual(t, "db25007f-7a18-43eb-9379-80131e44d633", usernoticeMessage.ID)
	assertFalse(t, usernoticeMessage.Action, "")
	assertIntsEqual(t, 0, len(usernoticeMessage.Emotes))
	assertStringsEqual(t, "resub", usernoticeMessage.MsgID)

	expectedParams := map[string]interface{}{
		"msg-param-cumulative-months":   6,
		"msg-param-streak-months":       2,
		"msg-param-should-share-streak": true,
		"msg-param-sub-plan":            "Prime",
		"msg-param-sub-plan-name":       "Prime",
	}
	assertStringInterfaceMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "ronni has subscribed for 6 months!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICEGiftSubMessage(t *testing.T) {
	testMessage := `@badges=staff/1,premium/1;color=#0000FF;display-name=TWW2;emotes=;id=e9176cd8-5e22-4684-ad40-ce53c2561c5e;login=tww2;mod=0;msg-id=subgift;msg-param-months=1;msg-param-recipient-display-name=Mr_Woodchuck;msg-param-recipient-id=89614178;msg-param-recipient-name=mr_woodchuck;msg-param-sub-plan-name=House\sof\sNyoro~n;msg-param-sub-plan=1000;room-id=19571752;subscriber=0;system-msg=TWW2\sgifted\sa\sTier\s1\ssub\sto\sMr_Woodchuck!;tmi-sent-ts=1521159445153;turbo=0;user-id=13405587;user-type=staff :tmi.twitch.tv USERNOTICE #forstycup`

	message := parseMessage(testMessage)
	_, usernoticeMessage := message.parseUSERNOTICEMessage()

	assertStringsEqual(t, "subgift", usernoticeMessage.MsgID)

	expectedParams := map[string]interface{}{
		"msg-param-months":                 1,
		"msg-param-recipient-display-name": "Mr_Woodchuck", // Maybe create a target User
		"msg-param-recipient-id":           "89614178",
		"msg-param-recipient-name":         "mr_woodchuck",
		"msg-param-sub-plan-name":          "House of Nyoro~n",
		"msg-param-sub-plan":               "1000",
	}
	assertStringInterfaceMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "TWW2 gifted a Tier 1 sub to Mr_Woodchuck!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICEAnonymousGiftSubMessage(t *testing.T) {
	testMessage := `@badges=broadcaster/1,subscriber/6;color=;display-name=qa_subs_partner;emotes=;flags=;id=b1818e3c-0005-490f-ad0a-804957ddd760;login=qa_subs_partner;mod=0;msg-id=anonsubgift;msg-param-months=3;msg-param-recipient-display-name=TenureCalculator;msg-param-recipient-id=135054130;msg-param-recipient-user-name=tenurecalculator;msg-param-sub-plan-name=t111;msg-param-sub-plan=1000;room-id=196450059;subscriber=1;system-msg=An\sanonymous\suser\sgifted\sa\sTier\s1\ssub\sto\sTenureCalculator!\s;tmi-sent-ts=1542063432068;turbo=0;user-id=196450059;user-type= :tmi.twitch.tv USERNOTICE #qa_subs_partner`

	message := parseMessage(testMessage)
	_, usernoticeMessage := message.parseUSERNOTICEMessage()

	assertStringsEqual(t, "anonsubgift", usernoticeMessage.MsgID)

	expectedParams := map[string]interface{}{
		"msg-param-months":                 3,
		"msg-param-recipient-display-name": "TenureCalculator", // Maybe create a target User
		"msg-param-recipient-id":           "135054130",
		"msg-param-recipient-user-name":    "tenurecalculator",
		"msg-param-sub-plan-name":          "t111",
		"msg-param-sub-plan":               "1000",
	}
	assertStringInterfaceMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "An anonymous user gifted a Tier 1 sub to TenureCalculator!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICERaidMessage(t *testing.T) {
	testMessage := `@badges=turbo/1;color=#9ACD32;display-name=TestChannel;emotes=;id=3d830f12-795c-447d-af3c-ea05e40fbddb;login=testchannel;mod=0;msg-id=raid;msg-param-displayName=TestChannel;msg-param-login=testchannel;msg-param-viewerCount=15;room-id=56379257;subscriber=0;system-msg=15\sraiders\sfrom\sTestChannel\shave\sjoined\n!;tmi-sent-ts=1507246572675;turbo=1;user-id=123456;user-type= :tmi.twitch.tv USERNOTICE #othertestchannel`

	message := parseMessage(testMessage)
	_, usernoticeMessage := message.parseUSERNOTICEMessage()

	assertStringsEqual(t, "raid", usernoticeMessage.MsgID)

	expectedParams := map[string]interface{}{
		"msg-param-displayName": "TestChannel",
		"msg-param-login":       "testchannel",
		"msg-param-viewerCount": 15,
	}
	assertStringInterfaceMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "15 raiders from TestChannel have joined!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICERitualMessage(t *testing.T) {
	testMessage := `@badges=;color=;display-name=SevenTest1;emotes=30259:0-6;id=37feed0f-b9c7-4c3a-b475-21c6c6d21c3d;login=seventest1;mod=0;msg-id=ritual;msg-param-ritual-name=new_chatter;room-id=6316121;subscriber=0;system-msg=Seventoes\sis\snew\shere!;tmi-sent-ts=1508363903826;turbo=0;user-id=131260580;user-type= :tmi.twitch.tv USERNOTICE #seventoes :HeyGuys`

	message := parseMessage(testMessage)
	_, usernoticeMessage := message.parseUSERNOTICEMessage()

	assertStringsEqual(t, "ritual", usernoticeMessage.MsgID)

	expectedParams := map[string]interface{}{
		"msg-param-ritual-name": "new_chatter",
	}
	assertStringInterfaceMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "Seventoes is new here!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERSTATEMessage(t *testing.T) {
	testMessage := `@badges=;color=#1E90FF;display-name=FletcherCodes;emote-sets=0,87321,269983,269986,568076,1548253;mod=0;subscriber=0;user-type= :tmi.twitch.tv USERSTATE #clippyassistant`

	message := parseMessage(testMessage)
	user, userstateMessage := message.parseUSERSTATEMessage()

	assertStringsEqual(t, "", user.ID)
	assertStringsEqual(t, "fletchercodes", user.Name)
	assertStringsEqual(t, "FletcherCodes", user.DisplayName)
	assertStringsEqual(t, "#1E90FF", user.Color)

	expectedBadges := map[string]int{}
	assertStringIntMapsEqual(t, expectedBadges, user.Badges)

	if userstateMessage.Type != USERSTATE {
		t.Error("parsing USERSTATE message failed")
	}
	assertStringsEqual(t, "USERSTATE", userstateMessage.RawType)
	assertStringsEqual(t, "", userstateMessage.Message)
	assertStringsEqual(t, "clippyassistant", userstateMessage.Channel)

	expectedEmoteSets := []string{"0", "87321", "269983", "269986", "568076", "1548253"}
	assertStringSlicesEqual(t, expectedEmoteSets, userstateMessage.EmoteSets)
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
