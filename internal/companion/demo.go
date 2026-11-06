package companion

const demoInstallation = "wisp-demo-v1"
const DemoCharacterID = demoInstallation + ":Demo-Ashenvale"

// Stable IDs and timestamps make activation repeatable without resetting conversations.
// These observations are invented, never claims about a real character or client.
func DemoExport() Export {
	out := Export{Schema: 1, InstallationID: demoInstallation, Character: Character{ID: "Demo-Ashenvale", Name: "Demospelare", Realm: "Ashenvale · testdata"}}
	for i, event := range []Event{
		{Kind: "snapshot", Zone: "Ashenvale", Subzone: "Maestra's Post", Level: 24},
		{Kind: "zone", Zone: "Ashenvale", Subzone: "Astranaar", Level: 24},
		{Kind: "note", Zone: "Ashenvale", Subzone: "Astranaar", Level: 24, Note: "Jag vill prata med Raene i Astranaar nästa gång. Det här är en påhittad demoanteckning."},
	} {
		event.ID = DemoCharacterID + ":" + string(rune('1'+i))
		event.ObservedAt = 1793871000 + int64(i*300)
		event.ClientVersion = "1.60.1"
		event.ClientBuild = "demo-not-observed"
		event.Interface = 16001
		out.Events = append(out.Events, event)
	}
	return out
}
