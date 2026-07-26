# Backend-Nachtloop — Treiber.
#
# Startet in einer eigenen Shell wiederholt `claude -p`. Jede Iteration ist ein
# frischer Prozess mit frischem Kontext; der Zustand liegt in BACKLOG.yml,
# JOURNAL.md und der Git-Historie. Deshalb degradiert der Lauf ueber eine Nacht
# nicht und ist beliebig verlaengerbar.
#
# Aufruf (aus dem Repo-Root):
#   powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -MaxIterations 2
#   powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -UntilTime "07:30"
#
# Abbrechen: Datei .planning\backend-block\loop\STOP anlegen (der Loop beendet
# nach der laufenden Iteration) oder Strg+C.
#
# Windows PowerShell 5.1: kein &&, kein ??, kein Ternary.

[CmdletBinding()]
param(
    [int]    $MaxIterations = 100,
    [string] $UntilTime     = "",        # "HH:mm", z.B. "07:30" — naechstes Auftreten
    [double] $BudgetUsd     = 12,        # Deckel pro Iteration
    [string] $Effort        = "high",
    [string] $ForceModel    = "",        # ueberschreibt das Modell aus BACKLOG.yml
    [switch] $DryRun                     # zeigt nur, was gestartet wuerde
)

$ErrorActionPreference = "Stop"

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

function Write-Line([string]$msg, [string]$color = "Gray") {
    Write-Host ("[{0}] {1}" -f (Get-Date -Format "HH:mm:ss"), $msg) -ForegroundColor $color
}

# --- Vorflug-Checks ----------------------------------------------------------
# Der Guard ist die einzige harte Grenze zum Production-Deploy. Laeuft er nicht
# oder ist er loechrig, startet hier nichts.
Write-Line "Vorflug: Guard-Regressionstest" "Cyan"
& bash ".planning/backend-block/loop/hooks/test-loop-guard.sh" | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Line "ABBRUCH: loop-guard.sh ist rot. Ohne Guard kein Lauf." "Red"
    exit 1
}

$branch = (& git rev-parse --abbrev-ref HEAD).Trim()
if ($branch -ne "backend-loop") {
    Write-Line "ABBRUCH: Branch ist '$branch', erwartet 'backend-loop'." "Red"
    exit 1
}

foreach ($f in @($Prompt, $Backlog)) {
    if (-not (Test-Path $f)) { Write-Line "ABBRUCH: fehlt: $f" "Red"; exit 1 }
}

if (Test-Path $StopFile) {
    Write-Line "STOP-Datei existiert noch vom letzten Lauf — wird entfernt." "Yellow"
    Remove-Item $StopFile -Force
}

# --- Deadline ----------------------------------------------------------------
$Deadline = [DateTime]::MaxValue
if ($UntilTime -ne "") {
    $Deadline = [DateTime]::ParseExact($UntilTime, "HH:mm", $null)
    if ($Deadline -le (Get-Date)) { $Deadline = $Deadline.AddDays(1) }
    Write-Line ("Deadline: {0}" -f $Deadline.ToString("yyyy-MM-dd HH:mm")) "Cyan"
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
$i = 0
$consecutiveFailures = 0
$rateLimitBackoffs = 0

Write-Line "Start. MaxIterations=$MaxIterations BudgetUsd=$BudgetUsd Effort=$Effort" "Green"

while ($i -lt $MaxIterations) {

    if (Test-Path $StopFile) { Write-Line "STOP-Datei gefunden — Lauf beendet." "Yellow"; break }
    if ((Get-Date) -ge $Deadline) { Write-Line "Deadline erreicht — Lauf beendet." "Yellow"; break }

    $open = Get-OpenUnitCount
    if ($open -eq 0) { Write-Line "Keine offenen Units mehr im Backlog — Lauf beendet." "Green"; break }

    $i++
    $model = Get-NextUnitModel
    $logFile = Join-Path $LogDir ("iter-{0:d3}.json" -f $i)

    Write-Line "--- Iteration $i / $MaxIterations  (Modell: $model, offen: $open) ---" "Cyan"

    if ($DryRun) {
        Write-Line "DryRun: wuerde starten mit Modell $model, Log $logFile" "Yellow"
        break
    }

    $started = Get-Date

    # --permission-mode bypassPermissions ist zwingend explizit: die globale
    # settings.json hat defaultMode "plan" — ohne Override wuerde die Iteration
    # nur planen statt bauen.
    & claude -p $promptText `
        --model $model `
        --permission-mode bypassPermissions `
        --settings $Settings `
        --effort $Effort `
        --max-budget-usd $BudgetUsd `
        --output-format json 2>&1 | Out-File -FilePath $logFile -Encoding utf8

    $exitCode = $LASTEXITCODE
    $elapsed  = [int]((Get-Date) - $started).TotalMinutes
    $body     = ""
    if (Test-Path $logFile) { $body = Get-Content $logFile -Raw }

    # Rate-Limit / Ueberlast: warten statt weiterbrennen.
    $isRateLimited = $false
    if ($body -match '(?i)(rate.?limit|usage limit|overloaded|429|resets at)') { $isRateLimited = $true }

    if ($isRateLimited) {
        $rateLimitBackoffs++
        if ($rateLimitBackoffs -gt 6) {
            Write-Line "Rate-Limit haelt an (6 Backoffs) — Lauf beendet." "Red"
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
        Write-Line "Iteration $i endete mit Exit $exitCode nach $elapsed min (Fehler in Folge: $consecutiveFailures)" "Red"
        if ($consecutiveFailures -ge 3) {
            Write-Line "Drei Fehlschlaege in Folge — Lauf beendet. Siehe $LogDir." "Red"
            break
        }
        Start-Sleep -Seconds 60
        continue
    }

    $consecutiveFailures = 0
    Write-Line "Iteration $i fertig nach $elapsed min. Log: $logFile" "Green"

    # Letzte Journal-Ueberschrift als Fortschrittsanzeige
    if (Test-Path $Journal) {
        $last = Select-String -Path $Journal -Pattern '^## Iteration' | Select-Object -Last 1
        if ($last) { Write-Line ("  " + $last.Line) "DarkGray" }
    }

    Start-Sleep -Seconds 15
}

Write-Line "Loop beendet nach $i Iteration(en)." "Green"
Write-Line "Review:  git log --oneline main..backend-loop" "Cyan"
Write-Line "Journal: $Journal" "Cyan"
