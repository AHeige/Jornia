# Lokal klientanalys — 6 november 2026

## Omfattning och metod

Ingen internetåtkomst användes i denna undersökning. Vi inventerade den installerade Forever-klientens appbundle, kataloger, cacheformat och filnamn samt läste utskrivbara strängar och importerade symboler ur klientens egen binär. Ingen klientbinär, cache, inställning eller spelprocess ändrades. Spelarnas chatlogg och inloggningsuppgifter lästes inte. Endast fältnamn ur chattkonfigurationen inventerades, inte dess innehåll.

Klient: `World of Warcraft.app`, 1.60.1.70790. Binärstorlek 124 829 424 byte. SHA-256: `4e262ff6f563f5f29819c6b06ad2bc48dc9a22579e35c36929c9298f3df9cbdf`. Råa strängutdrag och ett manifest ligger lokalt under ignorerade `.cache/client-file-audit/`; inga Blizzard-filer eller stora utdrag vendoreras.

Att ett namn finns i binären bevisar inte att en addon kan anropa funktionen, att funktionen är aktiverad i den här builden eller att den skriver direkt till disk.

## Starkaste nya spåret: klientens egen Lua-logg

Binären innehåller den exakta funktionssignaturen:

```lua
C_Log.LogMessage(message)
```

Samma API-grupp innehåller `LogWarningMessage`, `LogErrorMessage` och `LogMessageWithPriority`. Separat innehåller klienten:

- `developerLog`: beskrivs som aktivering av utvecklarloggen, standard av.
- `developerLogWriteToFile`: beskrivs som skrivning till fil, standard på.
- separata filter för bland annat normal-, varnings- och debugmeddelanden.
- filnamnet `DeveloperLog.log` och kodreferens till `UIUtil/DeveloperLog.cpp`.

Den faktiska filen finns under klientens `Logs`-katalog. Vid inventeringen var den 80 byte. Dess existens ensam bevisar inte att den tar emot användartext från Lua.

### Hypotes

En vanlig addon kan eventuellt skriva vår uttryckligt märkta text via `C_Log.LogMessage`, med filskrivning aktiverad. En lokal förstapartsprocess kan bevaka enbart dessa markörer. Det skulle ge:

```text
Wisp Lua → klientens native loggfunktion → DeveloperLog.log → lokal Wisp-process
```

Detta skulle inte behöva självviskningar, Blizzard-chatttrafik, processminnesåtkomst eller pixlar. Det sista påståendet gäller den föreslagna kedjans konstruktion, inte ett genomfört funktionstest. Vi har ännu inte visat att kedjan fungerar.

### Nästa enda test

1. Ladda vår egen inaktiva diagnostik-addon och kontrollera om `C_Log.LogMessage` faktiskt är en funktion i Lua-miljön. Läs aktuella värden för exakt de berörda utvecklarlogginställningarna.
2. Om den är tillgänglig: bevara ursprungsvärdena, aktivera endast nödvändiga logginställningar och skriv **en** unik, ofarlig teststräng. Ingen spelarkontext behövs för denna kontroll.
3. Bevaka endast rätt markör och mät synlighet på disk. Kontrollera även om klienten godtar eller nekar själva anropet. Ingen reload under intervallet.
4. Återställ inställningarna. Om ingen markör syns kan ett senare separat deltest pröva begränsad lokal utfyllnad för att undersöka buffring. Negativt resultat för ett kort meddelande är inte automatiskt bevis för att funktionen saknas.

Inga CVar-värden har ändrats och inget sådant test har startats i denna analys.

## Övriga fynd och bedömning

| Fynd i de lokala filerna | Vad det kan betyda | Vad det inte bevisar |
| --- | --- | --- |
| `ConsoleGetAllCommands`, `ConsoleEcho`, `ConsoleExec` | Klienten kan eventuellt själv redovisa tillåtna konsolkommandon och deras metadata. En riktad, läsande inventering i Lua är en möjlig fortsättning. | Konsolen är inte ett operativsystemsskal eller en godtycklig filskrivare. |
| `GxLogFlush` intill grafikens loggfunktioner och `gx.log` | Ett specifikt flushkommando kan finnas för grafikloggen. | Inte belägg för att det tömmer chatloggen eller utvecklarloggen; kalla det inte global flush. |
| `SetChatWindowName` och `chat-cache.txt` med `WINDOW`/`NAME` | Godtycklig text finns i en sparad chattinställning; en möjlig alternativ data-bärare. | Ingen explicit funktion för snabb chattkonfigurationssparning hittades. Den kan ha samma sparproblem som SavedVariables. |
| `SaveBindings`, `SetBinding` och `bindings-cache.wtf` | Det finns en explicit sparoperation för ett annat lokalt format. | Ingen allmän mailbox. Att ändra spelarens tangentbindningar är en olämplig normal produkttransport; eventuella hårdvaru-/combatkrav är oprövade. |
| `TestPrintToFile(snapshotID, filename)` intill textur-snapshotmetoder | Sannolikt ett grafik-/snapshotverktyg. | Inte belägg för godtycklig textskrivning. Vi har inte behandlat namnet som en sandbox-escape. |
| `questcache.wdb`, `creaturecache.wdb`, `gameobjectcache.wdb` | Innehåller bland annat questtexter och världsnamn som kan vara relevanta för senare lokal världskontext. | Inte en pålitlig ström av karaktärens aktuella questacceptans, progress eller frågor. |
| 48 GB i gemensamma `Data`-arkiv | Huvuddelen av installationen är spelresurser och index. | Stor datamängd innebär inte en tillgänglig IPC-kanal. Ingen fullständig 48 GB-extraktion gjordes. |

## Ett spår som ströks

Vid den första strängsökningen såg `SaveConfig` lovande ut. När API-gränserna kontrollerades visade det sig tillhöra **`C_ClassTalents`**, inte `C_ChatInfo`. Den föreslagna `C_ChatInfo.SaveConfig` är därför inte ett verifierat fynd och ska inte ligga till grund för implementation. Detta rättades innan något anrop eller inställningsändring gjordes.

## Varför buffertvolym fortfarande är relevant

I det föregående chatloggtestet blev den slutliga filen 3 810 byte, alltså mindre än 4 KiB. Det är förenligt med en volymstyrd buffert men bevisar varken en 4 KiB-gräns eller ett visst flushbeteende. Binären importerar bland annat `fwrite`, `fflush`, `setvbuf` och `fclose`; de är allmänna C-funktioner som kan användas på många håll i klienten och ger inget direkt Lua-anrop.

Vi ska inte skapa extra chatttrafik till spelservern för att fylla en okänd buffert som normal produktlösning. Om native Lua-loggningen fungerar kan ett begränsat, lokalt utfyllnadstest vara mer relevant: då testar vi bufferten utan att skicka spelarens data via en chattkanal. Det är fortfarande en senare hypotes.

## Beslut

Nästa föreslagna experiment är **native Lua-loggning med en enda testmarkör**. Det håller oss inom arbetssättet ett test i taget och angriper den saknade utgående transporten. Vår bevisade, begränsade fil-inbox för returvägen finns kvar. Ögonblicksbild och riktiga AI-svar påbörjas inte som del av denna filanalys.

## Senare klienttest

[Det genomförda enmarkörstestet](native-log-test-2026-11-06.md) verifierade att C_Log.LogMessage är tillgänglig och accepterar vår text. Den faktiska målfilen är **General.log**, inte den utvecklarlogg vi först antog. En korrekt bevakad upprepning skrev ut markören först i samband med klientavslut, efter cirka 189 sekunder. Fyndet ger alltså en lokal skrivväg men ännu ingen verifierad realtidskanal. Alla ändrade CVar-värden återställdes; nätåtkomst användes inte i testet.
