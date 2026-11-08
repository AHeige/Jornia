# Quest guidance from observed identity

## Player outcome

A player can ask what to do or where to turn a quest in without spelling its
English title. The per-send snapshot already contains numeric quest IDs. The
server now uses those identities to retrieve current-build world evidence,
including recipient and location relationships, alongside observed progress.
The same chat and consent preference remain in use; no client update is needed.

## Implementation

- `RetrieveQuestContext` constructs bounded, deduplicated ForeverDB quest anchors
  from active and recently turned-in quest IDs and the current imported build.
- Exact anchors rank ahead of lexical name matches. Existing two-hop graph
  retrieval, model context limits, source expiry, spoiler handling, provider
  pause and build isolation remain in force. No per-question provider calls.
- Ordinary questions without snapshots retain the existing text retrieval path.
  Quest-log-only mode retains its separately verified guidance path.
- Candidate guidance now distinguishes information absent from the client
  observation from world evidence available elsewhere. A ready quest is not
  already handed in. The model is instructed to use supplied turn-in relations
  and translate guidance into the question's language.

## Verification

All Go tests passed with a real, isolated PostgreSQL 16 test database, including
integration tests; `go vet ./...` passed. Identity tests cover Swedish, Chinese
and empty lexical input, recipient relationship expansion, deduplication,
expired sources, provider pause and build/game-version isolation. Service tests
confirm active and turned-in IDs reach identity retrieval.

A first VM inference check exposed contradictory candidate wording: it said the
recipient was absent even when world evidence supplied one. After correction,
three synthetic scenarios were sent through the real Go API and Berget:

| Scenario | Observed response | Backend time |
|---|---|---|
| Swedish next-step question, localized display title, quest 3901 with 0/12 progress | Named Rattling the Rattlecages and said to defeat 12 Rattlecage Skeletons | 1,397 ms |
| Chinese turn-in question, localized display title, quest 376 ready to hand in | `这个任务应该交给Novice Elreth。` with recipient entity and relationship citations | 1,749 ms |
| English remaining-items question, quest 376 explicitly turned in | Already handed in; no more Scavenger Paws needed for that completed run | 1,308 ms |

These are synthetic API/inference checks, not fresh observations of a player or
proof of the complete in-game transport. They establish neither language-wide
accuracy nor a general latency SLA. World citations retain `source_checked`
status, not independent in-game verification.

## Deployment and limits

VM API image: `wisp-private-api:20261108-quest-identity-2`. Only the API container
was recreated; all other pre-existing container IDs remained unchanged. Previous
image and environment backup are retained for rollback. No schema migration,
new dependency, addon reload, game input or gameplay automation was required.

Exact identity improves evidence selection; it does not create missing facts,
verify all follow-ups, solve conversational pronouns or calculate optimal routes.
World context remains bounded and may omit evidence in large quest logs.
A native chat walkthrough of this exact server revision remains to be observed.
