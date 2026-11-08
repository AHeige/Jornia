# Local computer and WoW client communication investigation

6 November 2026.

## Conclusion

**Direction after user guidance: prioritize SpeakText and the voice helper.** Only delivery during an uninterrupted session is relevant. Reload, logout and exit must not serve as transport, positive controls or required completion steps in subsequent tests. The log-volume route below is a hypothesis, not the selected next experiment.

The voice system's actual helper process/local transport is a separate lead, not yet a Wisp-readable channel or a finished localhost API.

No internet access/GitHub synchronization was used. The client was closed for analysis. No settings, client files, network connections or system protections changed. A separate control program operated only on our temporary files.

## Investigated material

- Installed universal Mach-O Forever 1.60.1.70790 executable: both architectures' strings, arm64 sections and system imports.
- `WowVoiceProxy` and helper files; no helper was started/replaced.
- Relationships between Lua functions, local formats, logs and internal services.
- Actual chat/native-log measurements and a new C-buffering control in **our own process**.

Distinguish functions present in the client, functions callable by addons, and transports readable by our external process.

## 1. Proven file route

```text
Addon Lua
  → C_Log.LogMessage
  → client logging / possible internal queue
  → process-local file buffer
  → operating-system filesystem
  → Wisp reads the matching marker
```

The exact split between client queue and C buffer is **unknown**. This shows possible layers, not a recovered complete call graph.

The earlier test established one accepted ordinary-Lua marker, actual target General.log, zero disk bytes during the interval/after reload, visibility only at exit after about 189 seconds, and final size 39,138 bytes. That rules out fast delivery for the tested small payload, not a volume threshold.

### What external fsync can and cannot do

Our small C process used fopen/fwrite. Separate Python checked size and called fsync on a separately opened descriptor. Both default buffering and explicit 64 KiB buffering were tested, never inside WoW.

| Own-process control | Small marker on disk | External fsync exposed it | More data exposed it |
|---|---|---|---|
| Default buffering on WoW's home filesystem | No | No | Yes, approximately 4 KiB blocks |
| Explicit 64 KiB buffer on home filesystem | No | No | Yes, after crossing the buffer |
| Default buffering on external work volume | No | No | Not within about 349 KiB; at close |
| Explicit 64 KiB buffer on external volume | No | No | Yes, after crossing the buffer |

Reported `st_blksize`: 4,096 bytes at home and 1,048,576 on the external volume. These are control behavior/metadata, not WoW buffering settings. Home-filesystem controls lived in our temporary Library/Caches directory and were deleted afterward; no player/game files were controls.

**Control conclusion:** external disk flushing cannot reach data still in another process's userspace buffer. File watchers, fsync or a different extension alone cannot solve that. Additional locally produced data may release older buffered data in some models.

General.log retained about 39 KiB while ordinary stdio on the same filesystem flushed at about 4 KiB. “Ordinary 4 KiB stdio” is therefore insufficient to explain it. Larger buffers, queues, other conditions or a different logging implementation remain possible.

### Untested hypothesis A: bounded local volume ramp

**Not performed in WoW; not the selected next test.**

- Use only verified C_Log.LogMessage and an independent General.log watcher.
- Begin with a short unique control marker.
- Increase synthetic local data in bounded cumulative steps, for example 1, 4, 16, 64 and at most 256 KiB. Include markers in an explicit total budget; never an unbounded loop.
- Spread small writes across frames for cancellation/UI responsiveness. Record submitted bytes/calls, errors and truncation; do not assume long messages survive intact.
- Measure which stage first exposes the original marker while the client stays open. Distinguish submitted bytes from actual log bytes.
- Keep other conditions comparable and preserve/restore relevant settings.

If successful, repeat and measure latency/jitter, CPU/disk cost, interruption and rotation before product use. If nothing arrives at the cap, stop; this does not prove impossibility for all volumes.

DeveloperLog's necessity for General.log is still untested. Do not change that simultaneously with volume: it mixes hypotheses.

## 2. New lead: the voice helper

The installation includes `World of Warcraft.app/Contents/Helpers/WowVoiceProxy.app`, with its own executable/audio libraries. Its binary contains:

- loopback address 127.0.0.1 and WebSocket-related code;
- listening, WoW-client connection and last-client-disconnected strings;
- `BlizzServiceRegistry` and `v0.blink.transport`;
- `VoiceService.rpc.cpp` and request/response type names;
- SpeakText, TTS status and local TTS handlers;
- Voice.log and a diagnostic format potentially containing TTS text.

The **game executable** contains:

```lua
C_VoiceChat.SpeakText(voiceID, text, rate, volume [, overlap])
```

This supports a hypothesis that ordinary voice functionality can convey Lua text to another local process. It is a more specific IPC clue than the word socket alone.

### Limits

- Live endpoint, port, authentication, encryption and exact wire format are unverified.
- A second connection does not automatically receive the first connection's data. This finding grants Jornia no voice-transport access.
- The diagnostic format does not prove current text logging, Voice.log as destination or fast writes.
- Provider/TTS availability can select different paths. Local and remote TTS are distinct.
- Replacing the proxy, changing the registry or capturing all voice/login traffic is inappropriate as a first test. None was done.

### Proposed experiment B: open-session delivery only

Start with read-only local TTS/voice inventory. If local TTS exists, use one synthetic phrase and independently time only its marker in voice diagnostics. No microphone, voice-channel join or collection of others' speech/chat is needed. Abort if local playback cannot be established; do not silently use remote TTS.

This may establish another outbound path, but external Jornia readability and buffering still need proof. Inspecting helper endpoint metadata without payloads is a separate possibility, not proof of an observable text channel.

## 3. Routes without a complete solution

| Route | Local evidence | Assessment |
|---|---|---|
| Generic localhost HTTP in Lua | No verified general call | Internal HTTP libraries do not give addons an HTTP API |
| Built-in browser/localhost URLs | Store/checkout/SSO code | Not an addon bridge; no URLs opened |
| PacketPipe | Socket/signaling/connection code | Internal transport, not a Lua file/socket API |
| mmap/shm_open/Mach messages | System imports exist | Internal capability, not Lua/external access to game data |
| SavedVariables | Verified persistence at save/reload/logout | Robust journal, not live transport |
| Bindings/configuration | SaveBindings/SetBinding/cache names | Risks player settings; save/combat restrictions untested; lower priority |
| Texture snapshot files | TestPrintToFile method name | No verified text API; user had also rejected pixels at this stage |
| External fsync/watchers | Negative own-process control for unserialized data | Cannot empty WoW userspace buffers |
| Read-only game memory | Earlier task-port request denied by macOS | No new attempts or protection changes |
| Unloaded addon inboxes | Two prior real deliveries | Limited inbound route, no outbound solution |

## Evidence and unknowns

Raw extracts/control results remain in ignored `.cache/client-file-audit/`, including `voice-proxy-strings.txt` and `own-process-buffer-control-filesystems.json`. No game binaries/libraries/logs are vendored or published. Controls use only C/Python standard libraries.

Static analysis established no reliable complete call graph/buffer size. Imports of fwrite/fflush/setvbuf cannot bind a buffer size to General.log. Analysis cannot replace a correct live test.

**User-directed order:** prioritize B; do not perform A here. No new live probe started. Steps 2/3 of that day's plan remain untouched.

## SpeakText log evidence

The helper contains a successful-TTS diagnostic format potentially carrying text, voice, destination and status. SDK destination names distinguish local playback and remote transmission. That motivates **local** SpeakText investigation, but neither destination file nor flush behavior is verified. No specific SpeakText.log file was found.

Local API material contains C_VoiceChat.GetTtsVoices(), with voiceID/name. Read-only function inspection and a nonempty local voice list must precede speech tests. SDK errors include missing/unsupported TTS engines; a signature does not establish working Mac TTS.

Next scope: inventory local voices → one synthetic phrase if local playback is confirmed → watch only its unique marker in appropriate client/helper logs while the game stays open. Set a short measurement deadline. No marker by that deadline means negative/incomplete live delivery; no exit/reload may rescue it. Microphone activation, channels, remote TTS, registry changes and helper replacement are excluded.
