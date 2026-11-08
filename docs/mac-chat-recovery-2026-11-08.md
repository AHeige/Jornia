# Mac chat recovery and composer behavior — 8 November 2026

## Scope

The user reported that sending a question did nothing, asked what happens when the chat is closed immediately after Enter, and requested that accepted messages disappear from the input immediately. They briefly authorized completing The Damned as a stale-state test, then explicitly stopped gameplay testing. No quest turn-in or post-turn-in answer was verified. Continue only chat/UI and read-only state work under the current authorization.

## Capture recovery

The local Go companion, QR reader and helper were running, and the local connection check reported the backend connected. WoW had a different process from the earlier successful laboratory session and initially appeared at the login screen. The old helper did not report the window/process it followed, so **the original missing send cannot be conclusively attributed to stale capture**.

The helper now selects the largest qualifying WoW application window instead of the first incidental window, reports its window/process ID (without titles, questions or screenshots), and rechecks window ownership and dimensions every two seconds while frames arrive. It stops the old stream and reacquires the current window after a change. A new attachment also resets the QR crop location. The reader resets capture readiness and its two-observation accumulator on attachment/detachment; it does not replay a paid question.

After a graceful restart of only the local reader, logs observed attachment to game process 7002/window 97408, detachment, then attachment to process 7044/window 97422. These process changes happened while the user was opening/logging into the game; no game restart/reload was initiated or required by the agent to complete the experiment.

A real question, **“How many Scavenger Paws do I still need?”**, subsequently delivered **“You need 1 more Scavenger Paw.”** The screenshot showed 5/6 paws, level 3 and a current approved snapshot, rather than the earlier 4/6 state. Request `941647961794164851000001` used Berget, 27.2 ms QR decoding, 4,147.5 ms return/11 blocks and 12,134.1 ms from accepted decoding to confirmed delivery. This verifies one current-state answer, not quest completion or general recovery reliability.

## Composer change

After the send hook returns an accepted operation ID:

- Add the user's question to the transcript.
- Keep the submitted question in the pending operation, scoped to its character.
- Clear the input and stored editable draft immediately; lock input while pending.
- Initially say **“Sending your message…”**, not that AI is already thinking. Progress reported by the transport persists when switching panel modes.
- Restore the submitted draft for the correct character on a later error. A send rejected before acceptance leaves the input unchanged. A success leaves it empty.

No animation is added: the new transcript entry, cleared composer and explicit pending status acknowledge the action without moving the layout or animating readable text. Existing controls and static QR behavior remain unchanged.

A one-session upgrade in an existing unused LoD slot was prepared and loaded without replacing the current view or clearing its messages. The complete source was also installed for future normal loading, with a backup. The slot file was restored after loading. No game reload was performed. Current-session visual verification of immediate clearing was still pending when computer-use input calls began timing out; do not substitute the unit test for that observation.

## Immediate-close limitation

The current QR protocol still needs a visible tile for two valid outbound observations and for return-block receipts. Hiding the chat before it has rendered/read can prevent submission. Hiding it after submission can stop return receipts even if the AI call succeeds and the local operation has a result. Current return receipt timeout is 12 seconds; the Lua operation timeout is 120 seconds, checked by the visible card's update loop.

The immediate-close experiment was requested but was **not yet observed end-to-end**. Computer-use calls to reopen the chat timed out; the user was asked to open `/jornia compact` for the remaining visual test. Subsequent screenshot-only calls also timed out after rebinding the application, so the current visual check remains blocked by the computer-use tool. No alternate UI-control mechanism or game reload was substituted. Do not call this reliable closed-window/background chat. A later implementation needs explicit submission acknowledgement and a bounded way to recover a saved answer after reopening, without duplicate paid inference or visible packets outside the approved chat tile. Hiding the QR alone is not a solution.

## Verification

The Objective-C capture helper compiled successfully with the existing Apple frameworks. The nine companion protocol tests passed. The affected Lua panel tests check immediate clearing, rejected-send preservation, error restoration, character isolation, duplicate-send locking and progress preservation across modes. Native transport tests also passed. Installed files preserve the existing journal/database.
