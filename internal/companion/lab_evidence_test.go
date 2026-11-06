package companion

import (
	"encoding/json"
	"os"
	"testing"
)

// Opt-in local forensic utility. Uses the bounded data-only reader, never a Lua
// interpreter. No player files are accessed by ordinary test runs.
func TestInspectChatProbeEvidence(t *testing.T) {
	source, output, run := os.Getenv("WISP_LAB_SOURCE"), os.Getenv("WISP_LAB_OUTPUT"), os.Getenv("WISP_LAB_RUN")
	if source == "" || output == "" || run == "" {
		t.Skip("explicit local evidence paths and run ID required")
	}
	b, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	object, err := savedObject(string(b), "WispBridgeLabDB", 4<<20)
	if err != nil {
		t.Fatal(err)
	}
	runs, ok := object["chat_runs"].([]any)
	if !ok {
		t.Fatal("no saved chat runs")
	}
	for _, value := range runs {
		entry, ok := value.(map[string]any)
		if !ok || entry["id"] != run {
			continue
		}
		data, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write(data); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err = file.Close(); err != nil {
			t.Fatal(err)
		}
		t.Log("Exported only the explicitly requested lab run to local JSON")
		return
	}
	t.Fatal("requested run not saved")
}
