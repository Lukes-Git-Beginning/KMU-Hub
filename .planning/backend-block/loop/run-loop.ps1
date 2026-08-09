# Backend-Nachtloop - Treiber.
#
# Startet in einer eigenen Shell wiederholt `claude -p`. Jede Iteration ist ein
# frischer Prozess mit frischem Kontext; der Zustand liegt in BACKLOG.yml,
# JOURNAL.md und der Git-Historie. Deshalb degradiert der Lauf ueber eine Nacht
# nicht und ist beliebig verlaengerbar.
#
# Aufruf (aus dem Repo-Root):
#   powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -MaxIterations 2
#   powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -UntilTime "07:30"
#   powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -UntilTime "10:00" -PauseFrom "20:30" -PauseTo "22:30"
#
# Abbrechen: Datei .planning\backend-block\loop\STOP anlegen (der Loop beendet
# nach der laufenden Iteration) oder Strg+C.
#
# Windows PowerShell 5.1: kein &&, kein ??, kein Ternary.

[CmdletBinding()]
param(
    [int]    $MaxIterations = 100,
    [int]    $StartAt       = 1,         # Iterationsnummer, bei der die Zaehlung beginnt
    [string] $UntilTime     = "",        # "HH:mm", z.B. "07:30" - naechstes Auftreten
    [string] $PauseFrom     = "",        # "HH:mm" - einmaliges Pausenfenster, Beginn
    [string] $PauseTo       = "",        # "HH:mm" - Ende (naechstes Auftreten nach PauseFrom)
    [int]    $PauseGuard    = 15,        # Minuten vor PauseFrom, ab denen keine Iteration mehr startet
    [double] $BudgetUsd     = 12,        # Deckel pro Iteration
    [string] $Effort        = "high",
    [string] $ForceModel    = "",        # ueberschreibt das Modell aus BACKLOG.yml
    [switch] $DryRun                     # zeigt nur, was gestartet wuerde
)

$ErrorActionPreference = "Stop"

# Die claude-CLI schreibt UTF-8. Ohne das dekodiert PowerShell ihre Ausgabe mit
# der OEM-Codepage (CP850) und in den iter-*.json landet Mojibake (verstuemmelte
# Umlaute und Sonderzeichen) - der JSON-Rahmen bleibt zwar parsebar, der
# Ergebnistext wird aber unlesbar.
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# --- Native Kommandos sicher aufrufen ----------------------------------------
# PS 5.1 verpackt JEDE stderr-Zeile eines nativen Exe in einen ErrorRecord.
# Zusammen mit $ErrorActionPreference = "Stop" wird daraus ein TERMINIERENDER
# Fehler - auch wenn das Programm mit Exit-Code 0 endet.
#
# Genau das ist am 2026-08-02 passiert: `git push` schreibt seinen Fortschritt
# ("To https://github.com/...") immer nach stderr. Der Push lief sauber durch,
# aber die Ausnahme riss die restliche CI-Phase mit - PR-Erkennung und
# CI-Wartelogik wurden nie erreicht, und 100 Commits lagen ungeprueft auf dem
# Branch, waehrend das Log wie ein Fehlschlag aussah.
#
# Deshalb: native Aufrufe immer hier durchreichen, und nie `2>&1` benutzen.
# Der Exit-Code ist die einzige verlaessliche Erfolgsaussage.
function Invoke-Native {
    param([Parameter(Mandatory)] [scriptblock] $Command)
    $prev = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try     { & $Command }
    finally { $ErrorActionPreference = $prev }
}

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")
$LoopDir  = Join-Path $RepoRoot ".planning\backend-block\loop"
$Prompt   = Join-Path $LoopDir "ITERATION.md"
$Backlog  = Join-Path $LoopDir "BACKLOG.yml"
$Journal  = Join-Path $LoopDir "JOURNAL.md"
$StopFile = Join-Path $LoopDir "STOP"
$LogDir   = Join-Path $LoopDir "logs"
$Settings = ".planning/backend-block/loop/loop-settings.json"

Set-Location $RepoRoot
if (-not (Test-Path $LogDir)) { New-Item -ItemType Directory -Path $LogDir | Out-Null }

$env:PATH = "$env:PATH;C:\Program Files\Go\bin;$env:USERPROFILE\go\bin"

$RunLog = Join-Path $LogDir "run.log"

# Konsolenzeilen zusaetzlich in eine Datei schreiben. Der Loop laeuft abgekoppelt
# in einem eigenen Fenster; ohne Mitschrift ist sein Fortschritt von aussen
# (Monitoring-Session, Wakeup-Check) nicht lesbar.
#
# Der catch darf NICHT stumm sein: haelt ein `tail -f` (oder ein verwaister
# tail-Prozess aus einem frueheren Lauf) die Datei offen, schlaegt jedes
# Add-Content fehl und der Fortschritt verschwindet spurlos. Im Lauf vom
# 2026-08-01 fehlten so die Iterationen 1-17 komplett im run.log. Einmalig
# warnen statt weiter ins Leere zu schreiben.
$script:LogWriteFailures = 0
function Write-Line([string]$msg, [string]$color = "Gray") {
    $line = "[{0}] {1}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss"), $msg
    Write-Host $line -ForegroundColor $color
    try {
        Add-Content -Path $RunLog -Value $line -Encoding utf8
        $script:LogWriteFailures = 0
    } catch {
        $script:LogWriteFailures++
        if ($script:LogWriteFailures -eq 1) {
            Write-Host "  [WARNUNG] run.log ist gesperrt - haelt ein tail-Prozess die Datei offen? Die Konsole bleibt die einzige Live-Sicht." -ForegroundColor Yellow
        }
    }
}

# --- Git Bash aufloesen ------------------------------------------------------
# Aus PowerShell heraus loest ein blankes `bash` auf WSL auf, nicht auf Git Bash
# ("execvpe(/bin/bash) failed"). Deshalb hier explizit den Git-Bash-Wrapper
# nehmen (bin\bash.exe, nicht usr\bin\bash.exe - letzterer hat die Utilities
# nicht im PATH).
$GitBash = ""
foreach ($c in @("C:\Program Files\Git\bin\bash.exe",
                 "C:\Program Files (x86)\Git\bin\bash.exe",
                 "$env:LOCALAPPDATA\Programs\Git\bin\bash.exe")) {
    if (Test-Path $c) { $GitBash = $c; break }
}
if ($GitBash -eq "") {
    Write-Line "ABBRUCH: Git Bash nicht gefunden. Der Guard-Test braucht bash." "Red"
    exit 1
}

# --- Vorflug-Checks ----------------------------------------------------------
# Der Guard ist die einzige harte Grenze zum Production-Deploy. Laeuft er nicht
# oder ist er loechrig, startet hier nichts.
Write-Line "Vorflug: Guard-Regressionstest" "Cyan"
# Durch Invoke-Native, weil sonst ausgerechnet der Fehlerpfad kaputt ist: ein
# roter Guard-Test schreibt seine Diagnose nach stderr, was unter
# ErrorActionPreference "Stop" eine Ausnahme wirft, bevor die
# LASTEXITCODE-Pruefung ueberhaupt erreicht wird - Abbruch mit
# NativeCommandError statt der klaren ABBRUCH-Meldung.
Invoke-Native { & $GitBash ".planning/backend-block/loop/hooks/test-loop-guard.sh" } | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Line "ABBRUCH: loop-guard.sh ist rot. Ohne Guard kein Lauf." "Red"
    exit 1
}

$branch = (Invoke-Native { & git rev-parse --abbrev-ref HEAD }).Trim()
if ($branch -ne "backend-loop") {
    Write-Line "ABBRUCH: Branch ist '$branch', erwartet 'backend-loop'." "Red"
    exit 1
}

foreach ($f in @($Prompt, $Backlog)) {
    if (-not (Test-Path $f)) { Write-Line "ABBRUCH: fehlt: $f" "Red"; exit 1 }
}

if (Test-Path $StopFile) {
    Write-Line "STOP-Datei existiert noch vom letzten Lauf - wird entfernt." "Yellow"
    Remove-Item $StopFile -Force
}

# --- Deadline ----------------------------------------------------------------
$Deadline = [DateTime]::MaxValue
if ($UntilTime -ne "") {
    $Deadline = [DateTime]::ParseExact($UntilTime, "HH:mm", $null)
    if ($Deadline -le (Get-Date)) { $Deadline = $Deadline.AddDays(1) }
    Write-Line ("Deadline: {0}" -f $Deadline.ToString("yyyy-MM-dd HH:mm")) "Cyan"
}

# --- Pausenfenster (einmalig) ------------------------------------------------
# Damit sich ein Nachtlauf abends ein paar Stunden aus dem Weg gehen kann (der
# Rechner wird gebraucht), ohne dafuer in zwei Laeufe zerlegt zu werden - zwei
# Laeufe heissen zwei Pushes, zwei CI-Laeufe und einen manuellen Neustart mitten
# am Abend.
#
# Bewusst EINMALIG statt taeglich wiederkehrend: die Grenzen werden hier zu
# absoluten Zeitpunkten aufgeloest, und das Fenster verbraucht sich, sobald es
# einmal gegriffen hat. Damit gibt es kein Mitternachts-Rollover zu behandeln und
# keinen Fall, in dem ein 14-Stunden-Lauf zweimal in dieselbe Pause faellt.
$PauseStart = [DateTime]::MaxValue
$PauseEnd   = [DateTime]::MaxValue
if (($PauseFrom -eq "") -xor ($PauseTo -eq "")) {
    Write-Line "ABBRUCH: -PauseFrom und -PauseTo gibt es nur zusammen." "Red"
    exit 1
}
if ($PauseFrom -ne "") {
    $PauseStart = [DateTime]::ParseExact($PauseFrom, "HH:mm", $null)
    if ($PauseStart -le (Get-Date)) { $PauseStart = $PauseStart.AddDays(1) }
    $PauseEnd = [DateTime]::ParseExact($PauseTo, "HH:mm", $null)
    while ($PauseEnd -le $PauseStart) { $PauseEnd = $PauseEnd.AddDays(1) }
    if ($PauseEnd -ge $Deadline) {
        Write-Line "ABBRUCH: Pausenende liegt auf oder hinter der Deadline - nach der Pause bliebe keine Arbeitszeit." "Red"
        exit 1
    }
    Write-Line ("Pause: {0} bis {1} (ab {2} min davor startet nichts Neues mehr)" -f `
        $PauseStart.ToString("yyyy-MM-dd HH:mm"), $PauseEnd.ToString("yyyy-MM-dd HH:mm"), $PauseGuard) "Cyan"
}

# --- Schlafsperre ------------------------------------------------------------
# Ohne das schickt Windows die Maschine mitten im Lauf in den Standby. Der Treiber
# und seine claude-Kindprozesse ueberleben das zwar (verifiziert 2026-08-01: der
# Lauf machte nach 47 Minuten Schlaf einfach weiter), aber die Nacht ist dann um
# die Schlafzeit kuerzer - bei einem 15-Stunden-Fenster kostet das echte Units.
#
# ES_CONTINUOUS | ES_SYSTEM_REQUIRED: System bleibt wach, das Display darf
# ausgehen. Die Anforderung haengt am Prozess und faellt weg, sobald er endet -
# deshalb ist das hier richtig aufgehoben und nicht als dauerhafte
# powercfg-Aenderung am System.
Add-Type -Name Power -Namespace Win32 -MemberDefinition @"
[DllImport("kernel32.dll", SetLastError = true)]
public static extern uint SetThreadExecutionState(uint esFlags);
"@
$ES_CONTINUOUS      = [uint32]"0x80000000"
$ES_SYSTEM_REQUIRED = [uint32]"0x00000001"
$sleepRc = [Win32.Power]::SetThreadExecutionState($ES_CONTINUOUS -bor $ES_SYSTEM_REQUIRED)
if ($sleepRc -eq 0) {
    Write-Line "WARNUNG: Schlafsperre konnte nicht gesetzt werden - der Lauf kann durch Standby unterbrochen werden." "Yellow"
} else {
    Write-Line "Schlafsperre aktiv (System bleibt wach, Display darf ausgehen)." "DarkGray"
}

# --- Modell der naechsten Unit aus BACKLOG.yml -------------------------------
# Bewusst simpel: erste 'model:'-Zeile nach der ersten 'status: todo'-Unit.
function Get-NextUnitModel {
    if ($ForceModel -ne "") { return $ForceModel }
    $lines = Get-Content $Backlog
    $currentModel = "sonnet"
    foreach ($line in $lines) {
        if ($line -match '^\s*-\s*id:')        { $currentModel = "sonnet" }
        if ($line -match '^\s*model:\s*(\S+)') { $currentModel = $Matches[1] }
        if ($line -match '^\s*status:\s*todo') { return $currentModel }
    }
    return "sonnet"
}

function Get-OpenUnitCount {
    $n = 0
    foreach ($line in (Get-Content $Backlog)) {
        if ($line -match '^\s*status:\s*(todo|in_progress)\s*$') { $n++ }
    }
    return $n
}

# --- Hauptschleife -----------------------------------------------------------
$promptText = Get-Content $Prompt -Raw
# Muss ein abgebrochener Lauf am selben Abend fortgesetzt werden (geaenderte
# Parameter, Rechnerneustart), zaehlt -StartAt weiter statt wieder bei 1 zu
# beginnen. Sonst ueberschreibt der zweite Lauf die iter-NNN.json des ersten und
# das JOURNAL.md fuehrt eine Nummer doppelt - genau der Fall, der die
# Fortschrittsanzeige unten (hoechste Journal-Nummer) schon einmal einfrieren liess.
$i = $StartAt - 1
$consecutiveFailures = 0
$rateLimitBackoffs = 0

# Iterationsnummer und Startzeit sind dem Modell sonst nicht bekannt - es raet.
# Ab Treiber-Iteration 27 schrieb es "## Iteration 28" ins Journal, seitdem lief
# die Zaehlung durchgehend um eins vor (eine Nummer doppelt, eine fehlte), und
# die Journal-Zeitstempel waren frei erfunden (Treiber 18:22, Journal "19:35").
# Der Treiber leitet seine Fortschrittsanzeige aus der hoechsten
# Journal-Iterationsnummer ab (siehe unten) - das ist also nicht kosmetisch.
#
# Einfach gequotetes Here-String (@'...'@), nicht doppelt gequotet: ein
# doppelt gequotetes wuerde den Backtick als Escape-Zeichen lesen. Einsetzen
# ueber den -f-Formatoperator, nicht ueber Interpolation.
$CtxTemplate = @'

---

## Laufkontext (vom Treiber gesetzt - nicht raten, nicht ableiten)

- Deine Iterationsnummer ist **{0}**.
- Startzeit dieser Iteration: **{1}**.

Beides gehoert unveraendert in die Kopfzeile deines Journal-Eintrags (Form siehe Schritt 6 oben).
Leite die Nummer nicht aus dem Journal ab und schaetze die Uhrzeit nicht.
'@

Write-Line "Start. MaxIterations=$MaxIterations BudgetUsd=$BudgetUsd Effort=$Effort" "Green"

while ($i -lt $MaxIterations) {

    if (Test-Path $StopFile) { Write-Line "STOP-Datei gefunden - Lauf beendet." "Yellow"; break }
    if ((Get-Date) -ge $Deadline) { Write-Line "Deadline erreicht - Lauf beendet." "Yellow"; break }

    $open = Get-OpenUnitCount
    if ($open -eq 0) { Write-Line "Keine offenen Units mehr im Backlog - Lauf beendet." "Green"; break }

    # Pausenfenster. Steht hinter dem Open-Count (sonst wuerde der Loop zwei
    # Stunden schlafen, obwohl der Backlog leer ist) und vor $i++ (die Pause ist
    # keine Iteration). Der Guard verhindert, dass kurz vor Pausenbeginn noch
    # eine ~15-min-Iteration losgeschickt wird und in die Pause hineinlaeuft -
    # eine laufende Iteration wird naemlich nicht abgebrochen.
    if ((Get-Date).AddMinutes($PauseGuard) -ge $PauseStart) {
        Write-Line ("Pause bis {0} - keine neue Iteration." -f $PauseEnd.ToString("HH:mm")) "Yellow"
        # In Minutenschritten schlafen, damit die STOP-Datei auch waehrend der
        # Pause wirkt - ein einzelnes Start-Sleep ueber zwei Stunden waere von
        # aussen nur noch per Strg+C zu beenden.
        while (((Get-Date) -lt $PauseEnd) -and (-not (Test-Path $StopFile))) {
            Start-Sleep -Seconds 60
        }
        $PauseStart = [DateTime]::MaxValue   # Fenster verbraucht
        $PauseEnd   = [DateTime]::MaxValue
        Write-Line "Pause vorbei." "Green"
        continue                              # STOP und Deadline oben neu pruefen
    }

    $i++
    $model = Get-NextUnitModel
    $logFile = Join-Path $LogDir ("iter-{0:d3}.json" -f $i)

    Write-Line "--- Iteration $i / $MaxIterations  (Modell: $model, offen: $open) ---" "Cyan"

    $iterPrompt = $promptText + ($CtxTemplate -f $i, (Get-Date -Format "yyyy-MM-dd HH:mm"))

    if ($DryRun) {
        Write-Line "DryRun: wuerde starten mit Modell $model, Log $logFile" "Yellow"
        Write-Line "DryRun: Laufkontext waere Iteration $i, Startzeit $(Get-Date -Format 'yyyy-MM-dd HH:mm')" "Yellow"
        break
    }

    $started = Get-Date

    # --permission-mode bypassPermissions ist zwingend explizit: die globale
    # settings.json hat defaultMode "plan" - ohne Override wuerde die Iteration
    # nur planen statt bauen.
    #
    # stderr geht in eine EIGENE Datei, nicht per 2>&1 in den JSON-Strom: sonst
    # wuerde eine einzelne Warnzeile der CLI das JSON zerschiessen (die
    # Rate-Limit-Erkennung unten liest dieses Feld) und - unter
    # ErrorActionPreference "Stop" - den ganzen Lauf abbrechen.
    $errFile = Join-Path $LogDir ("iter-{0:d3}.stderr.log" -f $i)

    Invoke-Native {
        & claude -p $iterPrompt `
            --model $model `
            --permission-mode bypassPermissions `
            --settings $Settings `
            --effort $Effort `
            --max-budget-usd $BudgetUsd `
            --output-format json 2>$errFile
    } | Out-File -FilePath $logFile -Encoding utf8

    $exitCode = $LASTEXITCODE
    # Leere stderr-Datei nicht als Muell im Logverzeichnis liegen lassen.
    if ((Test-Path $errFile) -and ((Get-Item $errFile).Length -eq 0)) {
        Remove-Item $errFile -Force -ErrorAction SilentlyContinue
    }
    $elapsed  = [int]((Get-Date) - $started).TotalMinutes
    $body     = ""
    if (Test-Path $logFile) { $body = Get-Content $logFile -Raw }

    # Rate-Limit / Ueberlast: warten statt weiterbrennen.
    #
    # Zwei Bedingungen, beide noetig. Der Lauf muss FEHLGESCHLAGEN sein - eine
    # erfolgreiche Iteration ist per Definition nicht rate-limitiert - und der
    # Text muss eine Limit-Formulierung enthalten.
    #
    # Warum so streng: die erste Fassung suchte '429' irgendwo im JSON und traf
    # damit die Ziffernfolge in Kosten-Floats und Tokenzahlen
    # ("total_cost_usd":5.367324299999998 enthaelt 429). Ergebnis war ein
    # 20-Minuten-Backoff nach einer voellig erfolgreichen Iteration. Deshalb
    # jetzt HTTP-429 nur noch in Feldkontext, und nie ohne Fehlerzustand.
    $isRateLimited = $false
    $runFailed = ($exitCode -ne 0) -or ($body -match '"is_error"\s*:\s*true')
    if ($runFailed -and $body -match '(?i)(rate.?limit|usage limit|overloaded|resets at|"status"\s*:\s*429|\b429\b)') {
        $isRateLimited = $true
    }

    if ($isRateLimited) {
        $rateLimitBackoffs++
        if ($rateLimitBackoffs -gt 6) {
            Write-Line "Rate-Limit haelt an (6 Backoffs) - Lauf beendet." "Red"
            break
        }
        Write-Line "Rate-Limit erkannt. Backoff 20 Minuten (Nr. $rateLimitBackoffs)." "Yellow"
        $i--                      # Iteration zaehlt nicht
        Start-Sleep -Seconds 1200
        continue
    }
    $rateLimitBackoffs = 0

    if ($exitCode -ne 0) {
        $consecutiveFailures++

        # Fehlerursache maschinenlesbar aus dem JSON-Body ziehen, statt nur Exit-
        # Code und Laufzeit zu loggen. Iteration 18 von Lauf 6 lief in den 12-USD-
        # Deckel (subtype "error_max_budget_usd", terminal_reason
        # "budget_exhausted") - im Log stand bisher nur "endete mit Exit 1 nach N
        # min", die eigentliche Ursache nur in der rohen iter-018.json.
        $subtype = "unbekannt"
        if ($body -match '"subtype"\s*:\s*"([^"]+)"')        { $subtype = $Matches[1] }
        $reason = "unbekannt"
        if ($body -match '"terminal_reason"\s*:\s*"([^"]+)"') { $reason = $Matches[1] }
        Write-Line "Iteration $i endete mit Exit $exitCode nach $elapsed min (Fehler in Folge: $consecutiveFailures, subtype=$subtype, terminal_reason=$reason)" "Red"

        # Jede fehlgeschlagene Iteration kann eine Unit auf in_progress
        # zurueckgelassen haben. Get-NextUnitModel und der Iterations-Prompt
        # ziehen ausschliesslich 'todo' - eine verwaiste in_progress-Unit waere
        # fuer den Rest des Laufs unsichtbar und bliebe liegen. Kein Set-Content:
        # BACKLOG.yml enthaelt zwar keine Umlaute, aber Set-Content wuerde
        # trotzdem eine BOM setzen, die den YAML-Parser stoert.
        $txt = [System.IO.File]::ReadAllText($Backlog)
        $new = [regex]::Replace($txt, '(?m)^(\s*status:\s*)in_progress\s*$', '${1}todo')
        if ($new -ne $txt) {
            [System.IO.File]::WriteAllText($Backlog, $new, (New-Object System.Text.UTF8Encoding($false)))
            Write-Line "  Haengende in_progress-Unit auf todo zurueckgesetzt." "Yellow"
        }

        if ($consecutiveFailures -ge 3) {
            Write-Line "Drei Fehlschlaege in Folge - Lauf beendet. Siehe $LogDir." "Red"
            break
        }
        Start-Sleep -Seconds 60
        continue
    }

    $consecutiveFailures = 0
    Write-Line "Iteration $i fertig nach $elapsed min. Log: $logFile" "Green"

    # Neueste Journal-Ueberschrift als Fortschrittsanzeige.
    #
    # Bewusst ueber die Iterationsnummer, NICHT ueber die Dateiposition: am
    # 2026-08-02 hat eine Iteration ihren Eintrag oberhalb des vorherigen
    # einsortiert (JOURNAL.md fuehrt "Iteration 37" vor "Iteration 36"), worauf
    # `Select-Object -Last 1` zwei Iterationen lang dieselbe Zeile meldete und
    # der Fortschritt stillzustehen schien.
    if (Test-Path $Journal) {
        $heads = Select-String -Path $Journal -Pattern '^## Iteration\s+(\d+)'
        if ($heads) {
            $newest = $heads | Sort-Object { [int]$_.Matches[0].Groups[1].Value } | Select-Object -Last 1
            Write-Line ("  " + $newest.Line) "DarkGray"
        }
    }

    Start-Sleep -Seconds 15
}

Write-Line "Loop beendet nach $i Iteration(en)." "Green"

# --- CI-Phase ----------------------------------------------------------------
# Genau ein Push am Ende des Laufs, und damit genau ein CI-Lauf, dessen Ergebnis
# morgens vorliegt. Bewusst hier im Treiber und nicht in der Iteration: der
# Guard blockt `git push` fuer den Agenten weiterhin hart, die Grenze bleibt
# also unveraendert. Der Treiber laeuft nicht unter dem Guard.
#
# Actions-Minuten sind der einzige echte Kostenposten (die beiden
# Review-Workflows laufen ueber CLAUDE_CODE_OAUTH_TOKEN, also das Abo). Ein Lauf
# pro Nacht ist ~10 Runner-Minuten - gegen einen Vormittag Fehlersuche ist das
# nichts.
#
# Kein `gh pr create`: das wuerde Claude PR Review und Security Review zuenden,
# beide ohne Draft-Gate. Existiert kein offener PR, wird nur gepusht und im Log
# vermerkt - den PR legt Luke bewusst an (Reihenfolge: erst beide Workflows
# disablen, dann PR oeffnen, nach gruenem CI wieder enablen).
if (-not $DryRun) {
    # ACHTUNG: JEDER native Aufruf in diesem Block muss durch Invoke-Native.
    # Empirisch geprueft (2026-08-02, PS 5.1): nicht nur `2>&1`, auch `2>$null`
    # terminiert unter ErrorActionPreference "Stop", sobald das Programm eine
    # stderr-Zeile schreibt - selbst bei Exit-Code 0. `gh` tut das regelmaessig
    # (Auth- und Deprecation-Hinweise). Dieser Block lief bis 2026-08-02 nie
    # vollstaendig durch und ist entsprechend ungehaertet gewesen.
    $ahead = Invoke-Native { & git rev-list --count "origin/backend-loop..HEAD" 2>$null }
    if ($LASTEXITCODE -ne 0) { $ahead = "0" }

    if ([int]$ahead -eq 0) {
        Write-Line "CI-Phase: keine neuen Commits - nichts zu pushen." "DarkGray"
    } else {
        Write-Line "CI-Phase: $ahead neue Commit(s), pushe backend-loop." "Cyan"
        # Kein `2>&1 | ForEach-Object`: git meldet den Push-Fortschritt ueber
        # stderr, was hier frueher die gesamte restliche CI-Phase abgeraeumt hat
        # (siehe Invoke-Native oben). Die Ausgabe geht direkt auf die Konsole,
        # der Exit-Code entscheidet.
        Invoke-Native { & git push origin backend-loop }

        if ($LASTEXITCODE -ne 0) {
            Write-Line "Push fehlgeschlagen - CI laeuft nicht. Commits liegen lokal." "Red"
        } else {
            $prState = Invoke-Native { & gh pr list --head backend-loop --state open --json number --jq ".[0].number" 2>$null }
            if ([string]::IsNullOrWhiteSpace($prState)) {
                Write-Line "Kein offener PR fuer backend-loop - CI wurde NICHT gestartet." "Yellow"
                Write-Line "  PR anlegen (Workflows vorher disablen!): siehe RESUME-Datei." "Yellow"
            } else {
                # Den Lauf ueber die gepushte SHA identifizieren, nicht ueber
                # `--limit 1`. Sonst wird der letzte, laengst abgeschlossene Lauf
                # gelesen und sofort als Ergebnis dieses Pushes gemeldet - ein
                # gruenes Signal, das nichts beweist. Zusaetzlich hat ci.yml einen
                # paths-Filter auf backend/**: eine Nacht, die nur Planning-Dateien
                # anfasst, startet gar keinen Lauf. Beides muss unterscheidbar sein
                # von "laeuft noch".
                $sha = (Invoke-Native { & git rev-parse HEAD }).Trim()
                Write-Line "Push gegen offenen PR #$prState - warte auf CI fuer $($sha.Substring(0,8))." "Cyan"

                $waited  = 0
                $runId   = ""
                $status  = ""
                # Erst bis zu 5 min auf den Start, dann bis zu 30 min auf das Ende.
                while ($waited -lt 2100) {
                    Start-Sleep -Seconds 60
                    $waited += 60
                    # Kein --jq mit eingebetteten Quotes ueber die Native-Grenze: PS 5.1
                    # frisst beim Weiterreichen an die native Exe das oeffnende \",
                    # schiebt ein Leerzeichen ein und macht aus dem schliessenden \" ein
                    # \) - jq bricht mit "invalid escape sequence" ab. Reproduziert in
                    # Lauf 6: der Treiber meldete "kein CI-Lauf fuer diese SHA", obwohl
                    # der Lauf 14 Sekunden nach dem Push existierte und gruen wurde.
                    # Deshalb hier das rohe JSON holen und selbst mit ConvertFrom-Json
                    # filtern (die drei anderen --jq-Aufrufe unten haben keine
                    # eingebetteten Quotes und sind davon nicht betroffen).
                    $runsJson = Invoke-Native { & gh run list --branch backend-loop --workflow ci.yml --limit 10 --json databaseId,headSha }
                    $runId = ""
                    if (-not [string]::IsNullOrWhiteSpace($runsJson)) {
                        # $runsJson kann ein String-Array sein (PS zerlegt mehrzeilige
                        # native Ausgabe in einzelne Zeilen) - vor ConvertFrom-Json mit
                        # -join zusammenfuegen, sonst bricht der Parser bei mehrzeiliger
                        # gh-Ausgabe.
                        #
                        # ConvertFrom-Json MUSS in einer eigenen Anweisung stehen, getrennt
                        # von Where-Object. Verifiziert (2026-08-09, PS 5.1): in EINER
                        # Pipeline verkettet (".. | ConvertFrom-Json | Where-Object {...}")
                        # reicht ConvertFrom-Json das komplette Array als EIN Pipeline-Objekt
                        # durch, statt es zu entrollen - Where-Object sieht dann in $_ das
                        # ganze Array. Die member-enumerierte Auswertung "$_.headSha -eq $sha"
                        # liefert dabei ein Array-Ergebnis statt Bool, ist damit immer
                        # wahr(-ig), und die komplette Liste rutscht ungefiltert durch.
                        $parsedRuns = ($runsJson -join "") | ConvertFrom-Json
                        $match = $parsedRuns | Where-Object { $_.headSha -eq $sha } | Select-Object -First 1
                        if ($match) { $runId = $match.databaseId }
                    }
                    if ([string]::IsNullOrWhiteSpace($runId)) {
                        if ($waited -ge 300) {
                            Write-Line "Nach 5 min kein CI-Lauf fuer diese SHA - paths-Filter (nur Nicht-Backend-Dateien geaendert) oder Lauf noch nicht sichtbar." "Yellow"
                            break
                        }
                        continue
                    }
                    $status = Invoke-Native { & gh run view $runId --json status --jq ".status" 2>$null }
                    if ($status -eq "completed") { break }
                }

                if ($status -eq "completed") {
                    $concl = Invoke-Native { & gh run view $runId --json conclusion --jq ".conclusion" 2>$null }
                    if ($concl -eq "success") {
                        Write-Line "CI GRUEN (Run $runId)." "Green"
                    } else {
                        Write-Line "CI ROT (Run $runId, $concl) - Details: gh run view $runId --log-failed" "Red"
                    }
                    # Ergebnis ins Journal, damit die erste Iteration der
                    # naechsten Nacht es im Verify-Vorspann sieht.
                    $stamp = Get-Date -Format "yyyy-MM-dd HH:mm"
                    $entry = "`n## CI nach Lauf ($stamp)`n- run: $runId`n- sha: $sha`n- ergebnis: $concl`n- commits: $ahead`n"
                    try { Add-Content -Path $Journal -Value $entry -Encoding utf8 } catch { }
                } elseif (-not [string]::IsNullOrWhiteSpace($runId)) {
                    Write-Line "CI (Run $runId) nach 35 min noch nicht fertig - morgens pruefen." "Yellow"
                }
            }
        }
    }
}

[Win32.Power]::SetThreadExecutionState($ES_CONTINUOUS) | Out-Null
Write-Line "Schlafsperre aufgehoben." "DarkGray"

Write-Line "Review:  git log --oneline main..backend-loop" "Cyan"
Write-Line "Journal: $Journal" "Cyan"
