package twitch

import (
	"testing"
)

func TestCantParseNoTagsMessage(t *testing.T) {
	testMessage := "my test message"

	message := ParseMessage(testMessage)
	rawMessage := message.(*RawMessage)

	if rawMessage.Type != UNSET {
		t.Errorf("parsing MessageType failed")
	}

	assertStringsEqual(t, "my", rawMessage.RawType)
	assertStringMapsEqual(t, nil, rawMessage.Tags)
	assertStringsEqual(t, "test message", rawMessage.Message)
}

func TestCantParseInvalidMessage(t *testing.T) {
	testMessage := "@my :test message"

	message := ParseMessage(testMessage)
	rawMessage := message.(*RawMessage)

	if rawMessage.Type != UNSET {
		t.Errorf("parsing MessageType failed")
	}

	assertStringsEqual(t, "message", rawMessage.RawType)

	expectedTags := map[string]string{
		"my": "",
	}
	assertStringMapsEqual(t, expectedTags, rawMessage.Tags)

	assertStringsEqual(t, "", rawMessage.Message)
}

func TestCantParsePartialMessage(t *testing.T) {
	testMessage := "@badges=;color=;display-name=ZZZi;emotes=;flags=;id=75bb6b6b-e36c-49af-a293-16024738ab92;mod=0;room-id=36029255;subscriber=0;tmi-sent-ts=1551476573570;turbo"

	message := ParseMessage(testMessage)
	rawMessage := message.(*RawMessage)

	if rawMessage.Type != UNSET {
		t.Errorf("parsing MessageType failed")
	}
	assertStringsEqual(t, "", rawMessage.RawType)

	expectedTags := map[string]string{
		"badges":       "",
		"color":        "",
		"display-name": "ZZZi",
		"emotes":       "",
		"flags":        "",
		"id":           "75bb6b6b-e36c-49af-a293-16024738ab92",
		"mod":          "0",
		"room-id":      "36029255",
		"subscriber":   "0",
		"tmi-sent-ts":  "1551476573570",
		"turbo":        "",
	}
	assertStringMapsEqual(t, expectedTags, rawMessage.Tags)
	assertStringsEqual(t, "", rawMessage.Message)
}

func TestCanParseSliceOutOfBoundsMessage(t *testing.T) {
	testMessage := "@badge-info=;badges=;color=#D2691E;display-name=xSpeedHack;emotes=245:38-52;flags=28-35:A.6/I.6;id=ebf30552-c327-4602-a346-582ecc880ab5;mod=0;room-id=23304775;subscriber=0;tmi-sent-ts=1556302007395;turbo=0;user-id=189609555;user-type= :xspeedhack!xspeedhack@xspeedhack.tmi.twitch.tv PRIVMSG #turntheslayer :Чет скушные катки Жек когда Хачаги?! ResidentSleeper"

	message := ParseMessage(testMessage)
	privateMessage := message.(*PrivateMessage)

	if privateMessage.Type != PRIVMSG {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "PRIVMSG", privateMessage.RawType)
	assertStringsEqual(t, "Чет скушные катки Жек когда Хачаги?! ResidentSleeper", privateMessage.Message)
	assertIntsEqual(t, 1, len(privateMessage.Emotes))
}

func TestCanParseWHISPERMessage(t *testing.T) {
	testMessage := "@badges=;color=#00FF7F;display-name=Danielps1;emotes=;message-id=20;thread-id=32591953_77829817;turbo=0;user-id=32591953;user-type= :danielps1!danielps1@danielps1.tmi.twitch.tv WHISPER gempir :i like memes"

	message := ParseMessage(testMessage)
	whisperMessage := message.(*WhisperMessage)
	user := whisperMessage.User

	assertStringsEqual(t, "32591953", user.ID)
	assertStringsEqual(t, "danielps1", user.Name)
	assertStringsEqual(t, "Danielps1", user.DisplayName)
	assertStringsEqual(t, "#00FF7F", user.Color)

	expectedBadges := map[string]int{}
	assertStringIntMapsEqual(t, expectedBadges, user.Badges)

	if whisperMessage.Type != WHISPER {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "WHISPER", whisperMessage.RawType)
	assertStringsEqual(t, "i like memes", whisperMessage.Message)
	assertIntsEqual(t, 0, len(whisperMessage.Emotes))
	assertFalse(t, whisperMessage.Action, "parsing Action failed")
}

func TestCanParseWHISPERActionMessage(t *testing.T) {
	testMessage := "@badges=;color=#1E90FF;display-name=FletcherCodes;emotes=;message-id=50;thread-id=269899575_408892348;turbo=0;user-id=269899575;user-type= :fletchercodes!fletchercodes@fletchercodes.tmi.twitch.tv WHISPER clippyassistant :/me tests whisper action"

	message := ParseMessage(testMessage)
	whisperMessage := message.(*WhisperMessage)

	assertTrue(t, whisperMessage.Action, "parsing Action failed")
}

func TestCanParsePRIVMSGMessage(t *testing.T) {
	type test struct {
		name            string
		message         string
		expectedMessage PrivateMessage
	}
	var tests = []test{
		{
			"Message With First Message",
			"@badges=premium/1;color=#DAA520;display-name=FletcherCodes;emotes=;first-msg=1;flags=;id=6efffc70-27a1-4637-9111-44e5104bb7da;mod=0;room-id=408892348;subscriber=0;tmi-sent-ts=1551473087761;turbo=0;user-id=269899575;user-type= :fletchercodes!fletchercodes@fletchercodes.tmi.twitch.tv PRIVMSG #clippyassistant :Chew your food slower... it's healthier",
			PrivateMessage{
				User: User{
					ID:          "269899575",
					Name:        "fletchercodes",
					DisplayName: "FletcherCodes",
					Color:       "#DAA520",
					Badges: map[string]int{
						"premium": 1,
					},
				},
				Type:         PRIVMSG,
				RawType:      "PRIVMSG",
				Message:      "Chew your food slower... it's healthier",
				Channel:      "clippyassistant",
				RoomID:       "408892348",
				ID:           "6efffc70-27a1-4637-9111-44e5104bb7da",
				FirstMessage: true,
			},
		},
		{
			"Message Without First Message",
			"@badges=premium/1;color=#DAA520;display-name=FletcherCodes;emotes=;first-msg=0;flags=;id=6efffc70-27a1-4637-9111-44e5104bb7da;mod=0;room-id=408892348;subscriber=0;tmi-sent-ts=1551473087761;turbo=0;user-id=269899575;user-type= :fletchercodes!fletchercodes@fletchercodes.tmi.twitch.tv PRIVMSG #clippyassistant :Chew your food slower... it's healthier",
			PrivateMessage{
				User: User{
					ID:          "269899575",
					Name:        "fletchercodes",
					DisplayName: "FletcherCodes",
					Color:       "#DAA520",
					Badges: map[string]int{
						"premium": 1,
					},
				},
				Type:         PRIVMSG,
				RawType:      "PRIVMSG",
				Message:      "Chew your food slower... it's healthier",
				Channel:      "clippyassistant",
				RoomID:       "408892348",
				ID:           "6efffc70-27a1-4637-9111-44e5104bb7da",
				FirstMessage: false,
			},
		},
		{
			"Message With Missing First Message",
			"@badges=premium/1;color=#DAA520;display-name=FletcherCodes;emotes=;flags=;id=6efffc70-27a1-4637-9111-44e5104bb7da;mod=0;room-id=408892348;subscriber=0;tmi-sent-ts=1551473087761;turbo=0;user-id=269899575;user-type= :fletchercodes!fletchercodes@fletchercodes.tmi.twitch.tv PRIVMSG #clippyassistant :Chew your food slower... it's healthier",
			PrivateMessage{
				User: User{
					ID:          "269899575",
					Name:        "fletchercodes",
					DisplayName: "FletcherCodes",
					Color:       "#DAA520",
					Badges: map[string]int{
						"premium": 1,
					},
				},
				Type:         PRIVMSG,
				RawType:      "PRIVMSG",
				Message:      "Chew your food slower... it's healthier",
				Channel:      "clippyassistant",
				RoomID:       "408892348",
				ID:           "6efffc70-27a1-4637-9111-44e5104bb7da",
				FirstMessage: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := ParseMessage(tt.message)
			privateMessage := message.(*PrivateMessage)
			user := privateMessage.User

			assertStringsEqual(t, tt.expectedMessage.User.ID, user.ID)
			assertStringsEqual(t, tt.expectedMessage.User.Name, user.Name)
			assertStringsEqual(t, tt.expectedMessage.User.DisplayName, user.DisplayName)
			assertStringsEqual(t, tt.expectedMessage.User.Color, user.Color)
			assertStringIntMapsEqual(t, tt.expectedMessage.User.Badges, user.Badges)

			if privateMessage.Type != tt.expectedMessage.Type {
				t.Error("parsing MessageType failed")
			}

			assertStringsEqual(t, tt.expectedMessage.RawType, privateMessage.RawType)
			assertStringsEqual(t, tt.expectedMessage.Message, privateMessage.Message)
			assertStringsEqual(t, tt.expectedMessage.Channel, privateMessage.Channel)
			assertStringsEqual(t, tt.expectedMessage.RoomID, privateMessage.RoomID)
			assertStringsEqual(t, tt.expectedMessage.ID, privateMessage.ID)
			assertBoolEqual(t, tt.expectedMessage.Action, privateMessage.Action)

			assertIntsEqual(t, len(tt.expectedMessage.Emotes), len(privateMessage.Emotes))
			assertIntsEqual(t, tt.expectedMessage.Bits, privateMessage.Bits)
			assertBoolEqual(t, tt.expectedMessage.FirstMessage, privateMessage.FirstMessage)
		})
	}

}

func TestCanParsePRIVMSGActionMessage(t *testing.T) {
	testMessage := "@badges=premium/1;color=#DAA520;display-name=FletcherCodes;emotes=;flags=;id=6efffc70-27a1-4637-9111-44e5104bb7da;mod=0;room-id=408892348;subscriber=0;tmi-sent-ts=1551473087761;turbo=0;user-id=269899575;user-type= :fletchercodes!fletchercodes@fletchercodes.tmi.twitch.tv PRIVMSG #clippyassistant :\u0001ACTION Thrashh5, FeelsWayTooAmazingMan kinda\u0001"

	message := ParseMessage(testMessage)
	privateMessage := message.(*PrivateMessage)

	assertTrue(t, privateMessage.Action, "parsing Action failed")
}

func TestCanParseEmoteMessage(t *testing.T) {
	testMessage := "@badges=;color=#008000;display-name=Zugren;emotes=120232:0-6,13-19,26-32,39-45,52-58;id=51c290e9-1b50-497c-bb03-1667e1afe6e4;mod=0;room-id=11148817;sent-ts=1490382458685;subscriber=0;tmi-sent-ts=1490382456776;turbo=0;user-id=65897106;user-type= :zugren!zugren@zugren.tmi.twitch.tv PRIVMSG #pajlada :TriHard Clap TriHard Clap TriHard Clap TriHard Clap TriHard Clap"

	message := ParseMessage(testMessage)
	privateMessage := message.(*PrivateMessage)

	assertIntsEqual(t, 1, len(privateMessage.Emotes))
	assertStringsEqual(t, "120232", privateMessage.Emotes[0].ID)
	assertStringsEqual(t, "TriHard", privateMessage.Emotes[0].Name)
	assertIntsEqual(t, 5, privateMessage.Emotes[0].Count)
	assertIntsEqual(t, 5, len(privateMessage.Emotes[0].Positions))
}

func TestCanHandleBadEmoteMessage(t *testing.T) {
	type test struct {
		name           string
		message        string
		expectedEmotes []Emote
	}
	var tests = []test{
		{
			"Broken Emote Message",
			"@badge-info=subscriber/3;badges=subscriber/3;color=#0000FF;display-name=Linkoping;emotes=25:40-44;flags=17-26:S.6;id=744f9c58-b180-4f46-bd9e-b515b5ef75c1;mod=0;room-id=188442366;subscriber=1;tmi-sent-ts=1566335866017;turbo=0;user-id=91673457;user-type= :linkoping!linkoping@linkoping.tmi.twitch.tv PRIVMSG #queenqarro :Då kan du begära skadestånd och förtal Kappa",
			[]Emote{{Name: "appa", ID: "25", Count: 1}},
		},
		{
			"Broken Emote Message",
			"@badge-info=;badges=moderator/1,partner/1;color=#5B99FF;display-name=StreamElements;emotes=86:30-39/822112:73-79;flags=22-27:S.5;id=03c3eec9-afd1-4858-a2e0-fccbf6ad8d1a;mod=1;room-id=506590738;subscriber=0;tmi-sent-ts=1588638345928;turbo=0;user-id=100135110;user-type=mod :streamelements!streamelements@streamelements.tmi.twitch.tv PRIVMSG #nobru :\u0001ACTION A LOJA AINDA NÃO ESTÁ PRONTA BibleThump , AGUARDE... NOVIDADES EM BREVE FortOne\u0001",
			[]Emote{{Name: "ibleThump ", ID: "86", Count: 1}, {Name: "ortOne", ID: "822112", Count: 1}},
		},
		{
			"Broken Emote Message",
			"@badge-info=subscriber/1;badges=subscriber/0;color=;display-name=jhoelsc;emotes=301683486:46-58,60-72,74-86/301683544:88-100;flags=0-4:S.6;id=1f1afcdd-d94c-4699-b35f-d214deb1e11a;mod=0;room-id=167189231;subscriber=1;tmi-sent-ts=1588640587462;turbo=0;user-id=505763008;user-type= :jhoelsc!jhoelsc@jhoelsc.tmi.twitch.tv PRIVMSG #staryuuki :pensé que no habría directo que bueno que si staryuukiLove staryuukiLove staryuukiLove staryuukiBits",
			[]Emote{{Name: "taryuukiLove ", ID: "301683486", Count: 3}, {Name: "taryuukiBits", ID: "301683544", Count: 1}},
		},
		{
			// This message is a modified example from https://github.com/twitchdev/issues/issues/104 that I have modified to make the emote be one extra character off, which I imagine could happen if the same unicode parsing thing magic bug happens twice in the same message
			"Extra Broken Emote Message",
			"@badge-info=subscriber/3;badges=subscriber/3;color=#0000FF;display-name=Linkoping;emotes=25:41-45;flags=17-26:S.6;id=744f9c58-b180-4f46-bd9e-b515b5ef75c1;mod=0;room-id=188442366;subscriber=1;tmi-sent-ts=1566335866017;turbo=0;user-id=91673457;user-type= :linkoping!linkoping@linkoping.tmi.twitch.tv PRIVMSG #queenqarro :Då kan du begära skadestånd och förtal Kappa",
			[]Emote{{Name: "ppa", ID: "25", Count: 1}},
		},
	}

	for _, tt := range tests {
		func(tt test) {
			t.Run(tt.name, func(t *testing.T) {
				message := ParseMessage(tt.message)
				privateMessage := message.(*PrivateMessage)

				assertIntsEqual(t, len(tt.expectedEmotes), len(privateMessage.Emotes))

				for i, expectedEmote := range tt.expectedEmotes {
					assertStringsEqual(t, expectedEmote.ID, privateMessage.Emotes[i].ID)
					assertStringsEqual(t, expectedEmote.Name, privateMessage.Emotes[i].Name)
					assertIntsEqual(t, expectedEmote.Count, privateMessage.Emotes[i].Count)
				}
			})
		}(tt)
	}
}

func TestCanParseBitsMessage(t *testing.T) {
	testMessage := "@badges=bits/5000;bits=5000;color=#007EFF;display-name=FletcherCodes;emotes=;flags=;id=405c4ccb-7d69-4a57-ac16-292e72ba288b;mod=0;room-id=408892348;subscriber=0;tmi-sent-ts=1551478518354;turbo=0;user-id=269899575;user-type= :fletchercodes!fletchercodes@fletchercodes.tmi.twitch.tv PRIVMSG #clippyassistant :showlove5000 Chew your food slower... it's healthier"

	message := ParseMessage(testMessage)
	privateMessage := message.(*PrivateMessage)

	assertIntsEqual(t, 5000, privateMessage.Bits)
}

func TestCanParseCLEARCHATMessage(t *testing.T) {
	testMessage := "@room-id=408892348;tmi-sent-ts=1551538661807 :tmi.twitch.tv CLEARCHAT #clippyassistant"

	message := ParseMessage(testMessage)
	clearchatMessage := message.(*ClearChatMessage)

	if clearchatMessage.Type != CLEARCHAT {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "CLEARCHAT", clearchatMessage.RawType)
	assertStringsEqual(t, "", clearchatMessage.Message)
	assertStringsEqual(t, clearchatMessage.Channel, "clippyassistant")
	assertStringsEqual(t, "408892348", clearchatMessage.RoomID)
}

func TestCanParseBanMessage(t *testing.T) {
	testMessage := "@room-id=408892348;target-user-id=269899575;tmi-sent-ts=1551538522968 :tmi.twitch.tv CLEARCHAT #clippyassistant :fletchercodes"

	message := ParseMessage(testMessage)
	clearchatMessage := message.(*ClearChatMessage)

	if clearchatMessage.Type != CLEARCHAT {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "CLEARCHAT", clearchatMessage.RawType)
	assertStringsEqual(t, "", clearchatMessage.Message)
	assertStringsEqual(t, clearchatMessage.Channel, "clippyassistant")
	assertStringsEqual(t, "408892348", clearchatMessage.RoomID)
	assertIntsEqual(t, 0, clearchatMessage.BanDuration)
	assertStringsEqual(t, "269899575", clearchatMessage.TargetUserID)
	assertStringsEqual(t, "fletchercodes", clearchatMessage.TargetUsername)
}

func TestCanParseTimeoutMessage(t *testing.T) {
	testMessage := "@ban-duration=5;room-id=408892348;target-user-id=269899575;tmi-sent-ts=1551538496775 :tmi.twitch.tv CLEARCHAT #clippyassistant :fletchercodes"

	message := ParseMessage(testMessage)
	clearchatMessage := message.(*ClearChatMessage)

	assertIntsEqual(t, 5, clearchatMessage.BanDuration)
}

func TestCanParseCLEARMSGMessage(t *testing.T) {
	testMessage := "@login=ronni;target-msg-id=abc-123-def :tmi.twitch.tv CLEARMSG #dallas :HeyGuys"

	message := ParseMessage(testMessage)
	clearMessage := message.(*ClearMessage)

	if clearMessage.Type != CLEARMSG {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "CLEARMSG", clearMessage.RawType)
	assertStringsEqual(t, "HeyGuys", clearMessage.Message)
	assertStringsEqual(t, "dallas", clearMessage.Channel)
	assertStringsEqual(t, "ronni", clearMessage.Login)
	assertStringsEqual(t, "abc-123-def", clearMessage.TargetMsgID)
}

func TestCanParseROOMSTATEMessage(t *testing.T) {
	testMessage := "@broadcaster-lang=en;emote-only=0;followers-only=-1;r9k=1;rituals=0;room-id=408892348;slow=0;subs-only=0 :tmi.twitch.tv ROOMSTATE #clippyassistant"

	message := ParseMessage(testMessage)
	roomstateMessage := message.(*RoomStateMessage)

	if roomstateMessage.Type != ROOMSTATE {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "ROOMSTATE", roomstateMessage.RawType)
	assertStringsEqual(t, "", roomstateMessage.Message)
	assertStringsEqual(t, "clippyassistant", roomstateMessage.Channel)
	assertStringsEqual(t, "408892348", roomstateMessage.RoomID)

	expectedState := map[string]int{
		"emote-only":     0,
		"followers-only": -1,
		"r9k":            1,
		"rituals":        0,
		"slow":           0,
		"subs-only":      0,
	}
	assertStringIntMapsEqual(t, expectedState, roomstateMessage.State)
}

func TestCanParseROOMSTATEChangeMessage(t *testing.T) {
	testMessage := `@followers-only=10;room-id=408892348 :tmi.twitch.tv ROOMSTATE #clippyassistant`

	message := ParseMessage(testMessage)
	roomstateMessage := message.(*RoomStateMessage)

	expectedState := map[string]int{
		"followers-only": 10,
	}
	assertStringIntMapsEqual(t, expectedState, roomstateMessage.State)
}

func TestCanParseUSERNOTICESubMessage(t *testing.T) {
	testMessage := "@badges=subscriber/0,premium/1;color=;display-name=FletcherCodes;emotes=;flags=;id=57cbe8d9-8d17-4760-b1e7-0d888e1fdc60;login=fletchercodes;mod=0;msg-id=sub;msg-param-cumulative-months=0;msg-param-months=0;msg-param-should-share-streak=0;msg-param-sub-plan-name=The\\sWhatevas;msg-param-sub-plan=Prime;room-id=408892348;subscriber=1;system-msg=fletchercodes\\ssubscribed\\swith\\sTwitch\\sPrime.;tmi-sent-ts=1551486064328;turbo=0;user-id=269899575;user-type= :tmi.twitch.tv USERNOTICE #clippyassistant"

	message := ParseMessage(testMessage)
	usernoticeMessage := message.(*UserNoticeMessage)
	user := usernoticeMessage.User

	assertStringsEqual(t, "269899575", user.ID)
	assertStringsEqual(t, "fletchercodes", user.Name)
	assertStringsEqual(t, "FletcherCodes", user.DisplayName)
	assertStringsEqual(t, "", user.Color)

	expectedBadges := map[string]int{
		"subscriber": 0,
		"premium":    1,
	}
	assertStringIntMapsEqual(t, expectedBadges, user.Badges)

	if usernoticeMessage.Type != USERNOTICE {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "USERNOTICE", usernoticeMessage.RawType)
	assertStringsEqual(t, "", usernoticeMessage.Message)
	assertStringsEqual(t, "clippyassistant", usernoticeMessage.Channel)
	assertStringsEqual(t, "408892348", usernoticeMessage.RoomID)
	assertStringsEqual(t, "57cbe8d9-8d17-4760-b1e7-0d888e1fdc60", usernoticeMessage.ID)
	assertIntsEqual(t, 0, len(usernoticeMessage.Emotes))
	assertStringsEqual(t, "sub", usernoticeMessage.MsgID)

	expectedParams := map[string]string{
		"msg-param-cumulative-months":   "0",
		"msg-param-months":              "0",
		"msg-param-should-share-streak": "0",
		"msg-param-sub-plan-name":       "The Whatevas",
		"msg-param-sub-plan":            "Prime",
	}
	assertStringMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "fletchercodes subscribed with Twitch Prime.", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICESubGiftMessage(t *testing.T) {
	testMessage := "@badges=subscriber/0,premium/1;color=#00FF7F;display-name=FletcherCodes;emotes=;flags=;id=b608909e-2089-4f97-9475-f2cd93f6717a;login=fletchercodes;mod=0;msg-id=subgift;msg-param-months=1;msg-param-origin-id=da\\s39\\sa3\\see\\s5e\\s6b\\s4b\\s0d\\s32\\s55\\sbf\\sef\\s95\\s60\\s18\\s90\\saf\\sd8\\s07\\s09;msg-param-recipient-display-name=NSFletcher;msg-param-recipient-id=418105091;msg-param-recipient-user-name=nsfletcher;msg-param-sender-count=0;msg-param-sub-plan-name=Channel\\sSubscription\\s(clippyassistant);msg-param-sub-plan=1000;room-id=408892348;subscriber=1;system-msg=FletcherCodes\\sgifted\\sa\\sTier\\s1\\ssub\\sto\\sNSFletcher!;tmi-sent-ts=1551487298580;turbo=0;user-id=79793581;user-type= :tmi.twitch.tv USERNOTICE #clippyassistant"

	message := ParseMessage(testMessage)
	usernoticeMessage := message.(*UserNoticeMessage)

	assertStringsEqual(t, "subgift", usernoticeMessage.MsgID)

	expectedParams := map[string]string{
		"msg-param-months":                 "1",
		"msg-param-origin-id":              "da 39 a3 ee 5e 6b 4b 0d 32 55 bf ef 95 60 18 90 af d8 07 09",
		"msg-param-recipient-display-name": "NSFletcher",
		"msg-param-recipient-id":           "418105091",
		"msg-param-recipient-user-name":    "nsfletcher",
		"msg-param-sender-count":           "0",
		"msg-param-sub-plan-name":          "Channel Subscription (clippyassistant)",
		"msg-param-sub-plan":               "1000",
	}
	assertStringMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "FletcherCodes gifted a Tier 1 sub to NSFletcher!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICEAnonymousGiftSubMessage(t *testing.T) {
	testMessage := `@badges=broadcaster/1,subscriber/6;color=;display-name=qa_subs_partner;emotes=;flags=;id=b1818e3c-0005-490f-ad0a-804957ddd760;login=qa_subs_partner;mod=0;msg-id=anonsubgift;msg-param-months=3;msg-param-recipient-display-name=TenureCalculator;msg-param-recipient-id=135054130;msg-param-recipient-user-name=tenurecalculator;msg-param-sub-plan-name=t111;msg-param-sub-plan=1000;room-id=196450059;subscriber=1;system-msg=An\sanonymous\suser\sgifted\sa\sTier\s1\ssub\sto\sTenureCalculator!\s;tmi-sent-ts=1542063432068;turbo=0;user-id=196450059;user-type= :tmi.twitch.tv USERNOTICE #qa_subs_partner`

	message := ParseMessage(testMessage)
	usernoticeMessage := message.(*UserNoticeMessage)

	assertStringsEqual(t, "anonsubgift", usernoticeMessage.MsgID)

	expectedParams := map[string]string{
		"msg-param-months":                 "3",
		"msg-param-recipient-display-name": "TenureCalculator", // Maybe create a target User
		"msg-param-recipient-id":           "135054130",
		"msg-param-recipient-user-name":    "tenurecalculator",
		"msg-param-sub-plan-name":          "t111",
		"msg-param-sub-plan":               "1000",
	}
	assertStringMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "An anonymous user gifted a Tier 1 sub to TenureCalculator!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICERaidMessage(t *testing.T) {
	testMessage := "@badges=partner/1;color=#00FF7F;display-name=FletcherCodes;emotes=;flags=;id=7a61cd41-f049-466b-9654-43e5bfc554aa;login=fletchercodes;mod=0;msg-id=raid;msg-param-displayName=FletcherCodes;msg-param-login=fletchercodes;msg-param-profileImageURL=https://static-cdn.jtvnw.net/jtv_user_pictures/herr_currywurst-profile_image-e6c037c9d321b955-70x70.jpeg;msg-param-viewerCount=538;room-id=269899575;subscriber=0;system-msg=538\\sraiders\\sfrom\\sFletcherCodes\\shave\\sjoined\\n!;tmi-sent-ts=1551490358542;turbo=0;user-id=269899575;user-type= :tmi.twitch.tv USERNOTICE #clippyassistant"

	message := ParseMessage(testMessage)
	usernoticeMessage := message.(*UserNoticeMessage)

	assertStringsEqual(t, "raid", usernoticeMessage.MsgID)

	expectedParams := map[string]string{
		"msg-param-displayName":     "FletcherCodes",
		"msg-param-login":           "fletchercodes",
		"msg-param-profileImageURL": "https://static-cdn.jtvnw.net/jtv_user_pictures/herr_currywurst-profile_image-e6c037c9d321b955-70x70.jpeg",
		"msg-param-viewerCount":     "538",
	}
	assertStringMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "538 raiders from FletcherCodes have joined!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICEUnraidMessage(t *testing.T) {
	testMessage := "@badges=broadcaster/1;color=#8A2BE2;display-name=FletcherCodes;emotes=;flags=;id=06e33f48-c728-4332-b4bc-b7eae6f59f3c;login=fletchercodes;mod=0;msg-id=unraid;room-id=269899575;subscriber=0;system-msg=The\\sraid\\shas\\sbeen\\scancelled.;tmi-sent-ts=1551518456143;turbo=0;user-id=269899575;user-type= :tmi.twitch.tv USERNOTICE #fletchercodes"

	message := ParseMessage(testMessage)
	usernoticeMessage := message.(*UserNoticeMessage)

	assertStringsEqual(t, "unraid", usernoticeMessage.MsgID)

	expectedParams := map[string]string{}
	assertStringMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "The raid has been cancelled.", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERNOTICERitualMessage(t *testing.T) {
	testMessage := "@badges=;color=;display-name=FletcherCodes;emotes=64138:0-8;flags=;id=e4090aa9-8079-41ff-904d-64c7a2193ee0;login=fletchercodes;mod=0;msg-id=ritual;msg-param-ritual-name=new_chatter;room-id=408892348;subscriber=0;system-msg=@FletcherCodes\\sis\\snew\\shere.\\sSay\\shello!;tmi-sent-ts=1551487438943;turbo=0;user-id=412636239;user-type= :tmi.twitch.tv USERNOTICE #clippyassistant :SeemsGood"

	message := ParseMessage(testMessage)
	usernoticeMessage := message.(*UserNoticeMessage)

	assertStringsEqual(t, "SeemsGood", usernoticeMessage.Message)
	assertStringsEqual(t, "ritual", usernoticeMessage.MsgID)

	expectedParams := map[string]string{
		"msg-param-ritual-name": "new_chatter",
	}
	assertStringMapsEqual(t, expectedParams, usernoticeMessage.MsgParams)

	assertStringsEqual(t, "@FletcherCodes is new here. Say hello!", usernoticeMessage.SystemMsg)
}

func TestCanParseUSERSTATEMessage(t *testing.T) {
	testMessage := "@badges=;color=#1E90FF;display-name=FletcherCodes;emote-sets=0,87321,269983,269986,568076,1548253;mod=0;subscriber=0;user-type= :tmi.twitch.tv USERSTATE #clippyassistant"

	message := ParseMessage(testMessage)
	userstateMessage := message.(*UserStateMessage)
	user := userstateMessage.User

	assertStringsEqual(t, "", user.ID)
	assertStringsEqual(t, "fletchercodes", user.Name)
	assertStringsEqual(t, "FletcherCodes", user.DisplayName)
	assertStringsEqual(t, "#1E90FF", user.Color)

	expectedBadges := map[string]int{}
	assertStringIntMapsEqual(t, expectedBadges, user.Badges)

	if userstateMessage.Type != USERSTATE {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "USERSTATE", userstateMessage.RawType)
	assertStringsEqual(t, "", userstateMessage.Message)
	assertStringsEqual(t, "clippyassistant", userstateMessage.Channel)

	expectedEmoteSets := []string{"0", "87321", "269983", "269986", "568076", "1548253"}
	assertStringSlicesEqual(t, expectedEmoteSets, userstateMessage.EmoteSets)
}

func TestCanParseNOTICEMessage(t *testing.T) {
	testMessage := "@msg-id=subs_on :tmi.twitch.tv NOTICE #clippyassistant :This room is now in subscribers-only mode."

	message := ParseMessage(testMessage)
	noticeMessage := message.(*NoticeMessage)

	if noticeMessage.Type != NOTICE {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "NOTICE", noticeMessage.RawType)
	assertStringsEqual(t, "This room is now in subscribers-only mode.", noticeMessage.Message)
	assertStringsEqual(t, "clippyassistant", noticeMessage.Channel)
	assertStringsEqual(t, "subs_on", noticeMessage.MsgID)
}

func TestCanParsePING1(t *testing.T) {
	testMessage := `PING :tmi.twitch.tv`
	rawMessage := ParseMessage(testMessage)
	message := rawMessage.(*PingMessage)

	assertStringsEqual(t, message.Message, "tmi.twitch.tv")
	assertMessageTypesEqual(t, PING, message.GetType())
}

func TestCanParsePING2(t *testing.T) {
	testMessage := `:tmi.twitch.tv PING :message`
	rawMessage := ParseMessage(testMessage)
	message := rawMessage.(*PingMessage)

	assertStringsEqual(t, message.Message, "message")
	assertMessageTypesEqual(t, PING, message.GetType())
}

func TestCanParsePING3(t *testing.T) {
	testMessage := `PING`
	rawMessage := ParseMessage(testMessage)
	message := rawMessage.(*PingMessage)

	assertStringsEqual(t, message.Message, "")
	assertMessageTypesEqual(t, PING, message.GetType())
}

func TestCanParsePING4(t *testing.T) {
	testMessage := `PING :message`
	rawMessage := ParseMessage(testMessage)
	message := rawMessage.(*PingMessage)

	assertStringsEqual(t, message.Message, "message")
	assertMessageTypesEqual(t, PING, message.GetType())
}

func TestCanParsePING5(t *testing.T) {
	testMessage := `PING :message anything after the first space should be ignored`
	rawMessage := ParseMessage(testMessage)
	message := rawMessage.(*PingMessage)

	assertStringsEqual(t, message.Message, "message")
	assertMessageTypesEqual(t, PING, message.GetType())
}

// potential other ping messages they could send according to the irc standard
// testMessage3 := `:tmi.twitch.tv PING :a b c` // reply a
// testMessage4 := `:tmi.twitch.tv PING` // reply PONG

func TestCanParsePONG1(t *testing.T) {
	testMessage := `:tmi.twitch.tv PONG tmi.twitch.tv :go-twitch-irc`
	rawMessage := ParseMessage(testMessage)
	message := rawMessage.(*PongMessage)

	assertStringsEqual(t, message.Message, "go-twitch-irc")
	assertMessageTypesEqual(t, PONG, message.GetType())
}

func TestCanParsePONG2(t *testing.T) {
	testMessage := `:tmi.twitch.tv PONG tmi.twitch.tv :go-twitch-irc lol`
	rawMessage := ParseMessage(testMessage)
	message := rawMessage.(*PongMessage)

	assertStringsEqual(t, message.Message, "go-twitch-irc")
	assertMessageTypesEqual(t, PONG, message.GetType())
}

func TestCanParsePONG3(t *testing.T) {
	testMessage := `PONG :tmi.twitch.tv`
	rawMessage := ParseMessage(testMessage)
	message := rawMessage.(*PongMessage)

	assertStringsEqual(t, message.Message, "")
	assertMessageTypesEqual(t, PONG, message.GetType())
}

func TestPRIVMSGEmotesParsedProperly(t *testing.T) {
	type test struct {
		name    string
		message string
	}
	var tests = []test{
		{
			"Normal PRIVMSG",
			"@badge-info=subscriber/52;badges=broadcaster/1,subscriber/48,partner/1;color=#CC44FF;display-name=pajlada;emotes=25:6-10/1902:16-20;flags=;id=2a3f9d35-5487-4239-80b3-6c9a5a1907a9;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1587291978478;turbo=0;user-id=11148817;user-type= :pajlada!pajlada@pajlada.tmi.twitch.tv PRIVMSG #pajlada :-tags Kappa 123 Keepo",
		},
		{
			"Action PRIVMSG",
			"@badge-info=subscriber/52;badges=broadcaster/1,subscriber/48,partner/1;color=#CC44FF;display-name=pajlada;emotes=25:6-10/1902:16-20;flags=;id=0c46c822-f668-4427-b19a-a1a0780a44ae;mod=0;room-id=11148817;subscriber=1;tmi-sent-ts=1587291881363;turbo=0;user-id=11148817;user-type= :pajlada!pajlada@pajlada.tmi.twitch.tv PRIVMSG #pajlada :\x01ACTION -tags Kappa 123 Keepo\x01",
		},
	}

	for _, tt := range tests {
		func(tt test) {
			t.Run(tt.name, func(t *testing.T) {
				message := ParseMessage(tt.message)
				privateMessage := message.(*PrivateMessage)
				assertIntsEqual(t, len(privateMessage.Emotes), 2)

				// Emote 1: Kappa at 6-10
				assertStringsEqual(t, privateMessage.Emotes[0].Name, "Kappa")
				assertStringsEqual(t, privateMessage.Emotes[0].ID, "25")

				// Emote 2: Keepo at 16-20
				assertStringsEqual(t, privateMessage.Emotes[1].Name, "Keepo")
				assertStringsEqual(t, privateMessage.Emotes[1].ID, "1902")
			})
		}(tt)
	}
}

func TestPRIVMSGMalformedEmotesDontCrash(t *testing.T) {
	type test struct {
		name    string
		message string
	}
	var tests = []test{
		{
			"Broken #1",
			"@badge-info=subscriber/52;badges=moderator/1,subscriber/48;color=#2E8B57;display-name=pajbot;emotes=80481_/3:7-14;flags=;id=1ec936d3-7853-4113-9984-664ac5c42694;mod=1;room-id=11148817;subscriber=1;tmi-sent-ts=1589640131796;turbo=0;user-id=82008718;user-type=mod :pajbot!pajbot@pajbot.tmi.twitch.tv PRIVMSG #pajlada :󠀀-tags pajaW_/3.0",
		},
	}

	for _, tt := range tests {
		func(tt test) {
			t.Run(tt.name, func(t *testing.T) {
				ParseMessage(tt.message)
			})
		}(tt)
	}
}

func TestCanParseGlobalUserStateMessage(t *testing.T) {
	testMessage := "@badge-info=;badges=;color=#2E8B57;display-name=pajbot;emote-sets=0,15961,24569,24570;user-id=82008718;user-type= :tmi.twitch.tv GLOBALUSERSTATE"

	message := ParseMessage(testMessage)
	globalUserStateMessage := message.(*GlobalUserStateMessage)
	user := globalUserStateMessage.User

	assertStringsEqual(t, "82008718", user.ID)
	assertStringsEqual(t, "pajbot", user.Name)
	assertStringsEqual(t, "pajbot", user.DisplayName)
	assertStringsEqual(t, "#2E8B57", user.Color)

	expectedBadges := map[string]int{}
	assertStringIntMapsEqual(t, expectedBadges, user.Badges)

	if globalUserStateMessage.Type != GLOBALUSERSTATE {
		t.Error("parsing MessageType failed")
	}
	assertStringsEqual(t, "GLOBALUSERSTATE", globalUserStateMessage.RawType)

	expectedEmoteSets := []string{"0", "15961", "24569", "24570"}
	assertStringSlicesEqual(t, expectedEmoteSets, globalUserStateMessage.EmoteSets)
}
