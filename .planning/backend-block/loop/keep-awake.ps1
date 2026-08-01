# Haelt das System wach, solange dieses Skript laeuft. Display darf ausgehen.
# Aufruf:  powershell -ExecutionPolicy Bypass -File keep-awake.ps1 -UntilTime "07:30"
param([string]$UntilTime = "07:30")

Add-Type -Name Power -Namespace Win32 -MemberDefinition @"
[DllImport("kernel32.dll", SetLastError = true)]
public static extern uint SetThreadExecutionState(uint esFlags);
"@

$ES_CONTINUOUS       = [uint32]"0x80000000"
$ES_SYSTEM_REQUIRED  = [uint32]"0x00000001"

$deadline = [DateTime]::ParseExact($UntilTime, "HH:mm", $null)
if ($deadline -le (Get-Date)) { $deadline = $deadline.AddDays(1) }

$r = [Win32.Power]::SetThreadExecutionState($ES_CONTINUOUS -bor $ES_SYSTEM_REQUIRED)
Write-Host ("[{0}] Schlafsperre aktiv (rc={1}), bis {2}" -f (Get-Date -Format "HH:mm:ss"), $r, $deadline.ToString("yyyy-MM-dd HH:mm"))

while ((Get-Date) -lt $deadline) {
    Start-Sleep -Seconds 60
    # Erneut anfordern: manche Energieprofile verwerfen die Anforderung nach einem Wake.
    [Win32.Power]::SetThreadExecutionState($ES_CONTINUOUS -bor $ES_SYSTEM_REQUIRED) | Out-Null
}

[Win32.Power]::SetThreadExecutionState($ES_CONTINUOUS) | Out-Null
Write-Host ("[{0}] Schlafsperre aufgehoben." -f (Get-Date -Format "HH:mm:ss"))