# Wisp: returkanal — kodgranskning och livekontroll, 6 november 2026

## Slutsats

Vi behöver skilja på **cachade filinnehåll** och **levande kontroll av filnärvaro**. Laddade Lua-moduler kan inte användas som återläsbara brevlådor i den nuvarande lösningen. Men en annan observerbar egenskap — om en känd fil fortfarande finns — kan ändras flera gånger och avläsas inne i spelet.

Efter granskning av andra projekts faktiska kod byggde vi en liten egen probe. På användarens Mac gav samma nollbytesfil **10 av/på-cykler och 20 observerade tillståndsbyten i Lua**, utan reload eller klientomstart. Detta bevisar en återanvändbar boolesk signal på denna klient. Det bevisar **inte ännu** en komplett eller snabb textkanal.

Rekommendation: testa först en egen liten filnärvarokanal för kontrollerade byte, därefter ett kort kontrollerat svar med CRC och kvittens. Behåll den fungerande OCR-vägen ut. Utöka inte blint mängden engångsaddoner.

## Vad granskades?

Publika källfiler hämtades till ignorerade `.cache/return-research-2026-11-06/`, med låsta Git-träd:

| Projekt | Granskat träd | Relevanta filer |
|---|---|---|
| `chelinho139/wow-ai` | `60165ca5bb6c6452f18604dc7b4fe8678b607bed` | `bridge/protocol.js`, `bridge/install-slots.js`, `addon/WoWAI/WoWAI.lua`, arkitekturdokument |
| `tommygeoco/neverquestalone` | `5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07` | `bridge/transport/signals.mjs`, `slots.mjs`, `publisher.mjs`, `addon/NeverQuestAlone/Transport.lua`, `tests/signals_test.mjs`, `docs/PROTOCOL.md` |
| `0xInuarashi/wow-forever-codex` | `85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8` | `companion/native.py`, `Native.lua`, `NativeProtocol.lua`, font-/cachetester och dokument |

Detta var läsning för mekanism och belägg, inte adoption. Ingen annans bibliotek, addon, app eller agentlösning installerades, kördes eller kopierades in i Wisp. Proben och pulskoden är våra egna.

Vår befintliga implementation granskades i `apps/ocr-lab/WispOCR/WispOCR.lua` och `scripts/experiments/ocr/bridge.py`: varje väntetick kan förbruka en ny LoD-modul; publicering skriver till alla 96 filer; varje sänd-/tänker-/slutstatus kan kräva ytterligare inläsning. Denna kod ändrades inte under researchen.

## 1. Den äldre ljudsignalen är inte samma sak som filnärvaro

I [wow-ai:s arkitektur](https://github.com/chelinho139/wow-ai/blob/60165ca5bb6c6452f18604dc7b4fe8678b607bed/docs/ARCHITECTURE.md#signals-the-empty-wav-trick) beskrivs tom → giltig WAV som en färdigflagga. Förbrukade ljudresurser kan förbli cachade. Projektet använder fortfarande 200 engångsmoduler och en vänteschemaläggning när signalerna inte fungerar. Fler moduler är en större ändlig buffert, inte återanvändning.

Min tidigare beskrivning av NeverQuestAlone som samma engångsmetod var för grov. Dess README beskrev en enklare bild, men den aktuella implementationen använder en annan mekanism. Den faktiska koden behöver väga tyngre än en sammanfattning.

## 2. NeverQuestAlone använder återkommande närvaropulser

I [signals.mjs, rader 1–20 och 88–128](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/bridge/transport/signals.mjs) är pulsen att **ta bort och återskapa samma redan kända fil**, inte att skriva nytt ljudinnehåll. Tomma filer används. Par av signalnamn minskar risken att en signal råkar saknas när gränssnittet laddas.

[Lua-mottagaren](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/addon/NeverQuestAlone/Transport.lua) anropar `PlaySoundFile`, stoppar returnerade handles och tolkar övergångar. En permanent kontrollfil skiljer saknad signal från t.ex. ett otillgängligt ljudsystem. Författarens [protokoll §3](https://github.com/tommygeoco/neverquestalone/blob/5eb98d5c33ad0c294144d5d1b0962a9fd2a70e07/docs/PROTOCOL.md#3-bridge--game-doorbells) hänför beteendet till Forever 1.60.1.70009.

Signalmodulen innehåller återställning och begränsade pulser. Returdata ligger ändå i ändliga addon-moduler: närvarosignalen talar om **när** en sådan ska laddas. Det projektet implementerar alltså inte automatiskt vår föreslagna textbuss nedan.

En hänvisad rå verifieringsfil, `docs/VERIFICATION.md` med E-015, gick inte att hämta på det granskade trädet. Kodkommentarer och tester är inte oberoende bevis för klientbeteendet. Därför genomförde vi den egna livekontrollen nedan.

## 3. Typsnitt kan bära text, men är fortfarande en filbank

[Font-skrivaren](https://github.com/0xInuarashi/wow-forever-codex/blob/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8/companion/native.py) kodar paket i teckenbredder. [Lua-mottagaren](https://github.com/0xInuarashi/wow-forever-codex/blob/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8/addon/CodexPixelBridge/Native.lua) använder `SetFont`, `SetText` och `GetStringWidth`, kalibrerar mätningen och kontrollerar paketen innan vanlig text visas. Detta ger datainnehåll utan en addon-modul per paket.

Deras [bankdokumentation](https://github.com/0xInuarashi/wow-forever-codex/blob/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8/docs/font-bank.md) beskriver 65 535 filnamn och hardlinks för att dela tomma basfiler. Atomiskt byte av en fil bryter dess länk; skrivning direkt genom länken skulle påverka andra namn. Filantal och metadata är fortfarande verkliga kostnader. Vi installerade inte denna bank.

[Cacheexperimentet](https://github.com/0xInuarashi/wow-forever-codex/blob/85953dbecbc2e8f766e3264d3f4ea1b01cfbe2f8/docs/font-recycling.md) gäller två diagnostikfiler på en äldre Forever-build, 1.60.1.69913. Det beskriver återanvändning efter ett annat klientprocessliv, **inte** återanvändning under ett oavbrutet spelpass eller automatisk återvinning av hela banken. Detta är reservspår för praktisk kapacitet, inte belägg för obegränsad retur.

## 4. Vår egen livekontroll

Klient: macOS, Forever 1.60.1.70790 / Interface 16001, Cone Thrower, PID 8007. Användaren gav uttrycklig tillåtelse att styra WoW. Spelet stod redan på karaktärsvalet; vi gick in på vald karaktär. Vi orsakade ingen utloggning, reload eller processomstart. Inga spelhandlingar utfördes.

Tre egna små testresurser lades i befintliga `WispBridgeLab/` medan karaktärsvalet visades, före inträdet i världen: två nollbytesfiler och en riktig tyst WAV. Detta fastställer inte upptäckt av nya namn under ett redan pågående inloggat UI.

Egen Lua-probe laddades via den förinstallerade, ännu oanvända `WispBridgeInbox04`. Detta förbrukade en plats **för att installera mätkoden**, inte en plats per signal. Proben gör inga filändringar, SavedVariables-skrivningar, AI-anrop eller inställningsändringar. Den visar returnerade värden direkt på skärmen, läser var 250 ms och slutar efter 30 sekunder.

Baslinje, visuellt observerad:

| Resurs | `PlaySoundFile` första returvärde |
|---|---|
| Fast tom kontrollfil | `true` |
| Tom pulsfil | `true` |
| Riktig tyst WAV | `true` |
| Befintlig egen labb-Lua | `true` |
| Befintlig egen labb-TOC | `true` |
| Saknad Lua-kontroll | `nil` |
| Saknad WAV-kontroll | `nil` |

Detta visar varför returvärdet inte kan tolkas som bevis för giltigt ljudinnehåll på denna klient. Inga ljudinställningar ändrades: `Sound_EnableAllSound` var 1.

Den externa pulskoden ändrade endast vår disponibla nollbytesfil `wispreturn_pulse.wav`: saknas i 0,5 s → finns i 0,5 s, tio gånger. Den återställdes i `finally`. Resultatet i det verkliga Lua-fönstret var **120 prover, 20 övergångar**. De andra kontrollvärdena var oförändrade. Samma filnamn användes hela tiden. Ingen ytterligare LoD-laddning behövdes för dessa övergångar.

En tidigare kort körning började nära den automatiska 30-sekundersgränsen: två av tre externa cykler hann observeras. Den användes inte som ett komplett trecykeltest. Den upprepade tiocykelkörningen genomfördes helt inom en ny mätperiod.

Efteråt återställdes inbox 04:s ursprungliga fil byte för byte, SHA-256 `86951c73c6c144f3e0ca568cc9822d0231303ff601fa2581262d89c0c0412277`. De tre egna probresurserna finns kvar för fortsatta tester. Lua-proben är stoppad efter 120 prover; `/wispreturn` kan starta en ny avgränsad period i detta UI. Inga produktionsjournaler eller hemligheter ändrades. Befintlig OCR/AI-kod ersattes inte.

Egen kod: `apps/bridge-lab/return-presence-probe.lua`, `scripts/experiments/pulse_return_presence.py`. Råa externa pulstider och sammanfattad observation finns i ignorerad `.cache/return-research-2026-11-06/wisp-probe/`. Det är två olika klockor, så detta är **inte** en exakt envägslatensmätning. 250 ms är samplingsintervallet, inte en uppmätt latensgaranti.

## 5. Föreslagen återanvändbar textkanal — ännu inte implementerad

**Vår härledning från signalprovet:** om samma observerbara booleska filnärvaro kan ändras upprepade gånger, kan flera sådana filer representera bitar i ett datapaket. Lua rekonstruerar då själva svarstexten utan LoD-inboxar. NQA-koden visar dörrklockor, inte denna textöverföring.

Exempel på ett litet test: åtta återanvändbara datafiler motsvarar en byte; en fast kontroll visar att kanalen är läsbar; separat giltighets-/sekvensinformation hindrar mottagaren från att läsa mitt i en filuppdatering. Datorappen lämnar paketet oförändrat tills Lua visar en kvittens som vår OCR kan läsa. Kvittensen måste knytas till session, karaktär, fråga och paketnummer.

Avsändaren får inte behandla en atomisk filoperation som en atomisk ändring av hela filgruppen. Därför behövs publiceringsordning, fullständig checksumma och omsändning vid avslag. Starta med en liten sekvens av kontrollerade byte, inklusive nollor och ettor; bygg inte direkt en AI-integration.

Krav för att kunna kalla det texttransport:

1. Samma filuppsättning ska leverera olika byte flera gånger, utan nya namn, reload eller omstart.
2. Paket med fel kontrollsumma, gammal session, fel karaktär eller fel request-ID ska avvisas. Samma kvittens får inte skapa dubbla meddelanden eller betalda AI-anrop.
3. Tappad skärmfångst, felordnade filuppdateringar, låg bildfrekvens och pausad/otillgänglig ljudkanal ska ge väntan eller avslag, inte falska nollbitar.
4. Uppmätt API-/filkostnad, bytes/sekund och påverkan på bildfrekvens ska styra antal parallella bitfiler och läsintervall. Sju läsningar per prov i denna kontroll säger inget om kostnaden för exempelvis 256 samtidiga filer.
5. Fråga → svar ska därefter testas från början till slut. Enstaka signalbyten är inte samma sak som ett färdigt svar.

Teoretisk nyttolast före protokollkostnad: `N datafiler / 8` byte per stabil avläsning. Vid 64 filer är det 8 byte; vid 256 filer 32 byte. Faktisk hastighet blir lägre när skrivning, kontroll, kvittens, omsändning och UI-tidsbudget räknas in. Dessa tal är kapacitet per avläsning, **inte en benchmark eller ett löfte om hastighet**.

## 6. Beslut efter researchen

| Spår | Bevisnivå för Wisp | Nästa användning |
|---|---|---|
| Nuvarande LoD-inboxar | Riktigt AI-svar verifierat; ändlig pool | Bevara fungerande referens |
| Återanvändbar filnärvaro | Tio cykler / tjugo övergångar på aktuell Mac | Nästa kandidat för egen byte-/texttransport |
| Filnärvaro som enbart färdigsignal | Signalprimitiven verifierad, integration saknas | Kan minska tomma LoD-läsningar, men tar inte bort deras ändlighet |
| Typsnittsbank | Kod och andra författares rapporter granskade | Reservspår; ingen Wisp-liveverifiering eller installation |
| Omladdning av samma Lua-inbox | Ingen återanvändningsmekanism identifierad i granskad kod/API-yta | Ska inte utlovas |

Den avgörande nya möjligheten är alltså **återanvändbar observerbar filnärvaro**, inte att återställa addon-laddaren. Vi har hittat och verifierat en signalprimitiv som kan vara grund för returen. Uthållig, snabb och tillförlitlig svarstext kräver fortfarande nästa protokollexperiment.
