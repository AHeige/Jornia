# Local client-file audit on 6 November 2026

## Scope and method

This investigation used no internet access. We inventoried the installed Forever app bundle, directories, cache formats and filenames, and read printable strings/imported symbols from its own executable. No client binary, cache, setting or game process changed. Player chat logs and credentials were not read. Only chat-configuration field names were inventoried, not their contents.

Client: `World of Warcraft.app`, 1.60.1.70790. Binary size: 124,829,424 bytes. SHA-256: `4e262ff6f563f5f29819c6b06ad2bc48dc9a22579e35c36929c9298f3df9cbdf`. Raw strings/manifest remain locally under ignored `.cache/client-file-audit/`; no Blizzard files or large extracts are vendored.

A name in a binary does not establish addon-callability, activation in this build or immediate disk writing.

## Strongest lead: native Lua logging

The binary contains the exact signature:

```lua
C_Log.LogMessage(message)
```

The API group also contains `LogWarningMessage`, `LogErrorMessage` and `LogMessageWithPriority`. Separately the client contains:

- `developerLog`: described as enabling developer logging, off by default.
- `developerLogWriteToFile`: file output, on by default.
- Separate normal/warning/debug filters.
- `DeveloperLog.log` and a `UIUtil/DeveloperLog.cpp` reference.

The actual file existed under `Logs`, then 80 bytes. Existence alone does not prove it receives user text from Lua.

### Hypothesis

An ordinary addon might write explicitly labeled text through C_Log.LogMessage with file output enabled; a first-party local process could watch only those markers:

```text
Wisp Lua → native client logging → DeveloperLog.log → local Wisp process
```

The proposed design needs no self-whispers, Blizzard chat traffic, process-memory access or pixels. That describes the hypothesis, not a completed test; the chain had not yet been shown to work.

### Next bounded test

1. Load our inert diagnostic addon; check that C_Log.LogMessage is a function and read only the relevant original logging settings.
2. If available, preserve original values, enable only necessary settings and write **one** harmless unique marker, with no player context.
3. Watch only that marker and measure disk visibility; record acceptance/rejection. No reload during the interval.
4. Restore settings. If unseen, a later separate bounded local-filler test could investigate buffering. A short-message failure does not prove the function absent.

No CVar changed and no such test started during this file audit.

## Other findings

| Finding | Possible meaning | Does not establish |
|---|---|---|
| `ConsoleGetAllCommands`, `ConsoleEcho`, `ConsoleExec` | The client may enumerate allowed console commands/metadata; targeted read-only Lua inventory could follow | The console is not an OS shell or arbitrary file writer |
| `GxLogFlush` near graphics logging and `gx.log` | A graphics-specific flush command may exist | No evidence of chat/developer flush; do not call it global |
| `SetChatWindowName`, `chat-cache.txt` with `WINDOW`/`NAME` | Arbitrary text exists in saved chat configuration, a possible carrier | No explicit fast configuration-save function found; may share SavedVariables delays |
| `SaveBindings`, `SetBinding`, `bindings-cache.wtf` | An explicit save operation exists for another format | Not a general mailbox; altering player bindings is inappropriate normal transport, and hardware/combat restrictions are untested |
| `TestPrintToFile(snapshotID, filename)` near texture snapshots | Likely graphics/snapshot tooling | No proof of arbitrary text output; not treated as a sandbox escape |
| `questcache.wdb`, `creaturecache.wdb`, `gameobjectcache.wdb` | Quest text/world names might support later local world context | Not a reliable stream of current quest acceptance, progress or questions |
| 48 GB shared `Data` archives | Most installation space is resources/indexes | Data volume is not IPC availability; no full 48 GB extraction performed |

## A discarded lead

`SaveConfig` initially looked promising. Checking API boundaries showed it belonged to **C_ClassTalents**, not C_ChatInfo. The proposed C_ChatInfo.SaveConfig is therefore not a verified finding and must not drive implementation. This was corrected before any call or setting change.

## Why buffer volume remains relevant

The previous final chat log was 3,810 bytes, below 4 KiB. That is consistent with volume-triggered buffering but proves neither a 4 KiB threshold nor a flush mechanism. Imported C functions `fwrite`, `fflush`, `setvbuf` and `fclose` may be used throughout the client; they expose no direct Lua call.

Do not generate extra server chat traffic to fill an unknown buffer as a normal product solution. If native logging works, bounded local filler might test buffering without sending player data through chat. This remained a later hypothesis.

## Decision

Next experiment: **one-marker native Lua logging**. This preserves one-test-at-a-time work and targets the missing outbound path. The limited proven return inbox remains available. This file audit does not start snapshots or real AI replies.

## Subsequent client test

The [one-marker test](native-log-test-2026-11-06.md) verified C_Log.LogMessage availability/acceptance. Actual destination: **General.log**, not the assumed developer log. A correctly watched repeat exposed the marker only at exit after about 189 seconds. This establishes local writing, not live transport. All changed CVars were restored; the test used no network access.
