param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$StopFile = ".\stop-postman-loop.txt",
    [string]$LogFile = ".\runs\postman-loop.log"
)

New-Item -ItemType Directory -Force (Split-Path -Parent $LogFile) | Out-Null
Remove-Item $StopFile -Force -ErrorAction SilentlyContinue
$iteration = 0
$payload = @{
    sessionId = "postman-loop"
    msisdn = "2348031000001"
    network = "MTN"
    ussdString = "*737#"
    input = ""
    stage = "begin"
} | ConvertTo-Json -Compress

while (-not (Test-Path $StopFile)) {
    $iteration++
    $timestamp = Get-Date -Format "o"
    try {
        $health = Invoke-WebRequest -Uri "$BaseUrl/healthz" -UseBasicParsing -TimeoutSec 15
        $receive = Invoke-WebRequest -Uri "$BaseUrl/api/v1/receive" -Method Post -ContentType "application/json" -Body $payload -UseBasicParsing -TimeoutSec 15
        $send = Invoke-WebRequest -Uri "$BaseUrl/api/v1/send" -Method Post -ContentType "application/json" -Body $payload -UseBasicParsing -TimeoutSec 15
        "$timestamp iteration=$iteration health=$($health.StatusCode) receive=$($receive.StatusCode) send=$($send.StatusCode)" | Tee-Object -FilePath $LogFile -Append
    } catch {
        "$timestamp iteration=$iteration error=$($_.Exception.Message)" | Tee-Object -FilePath $LogFile -Append
    }
}

"$(Get-Date -Format o) stopped=true iterations=$iteration" | Tee-Object -FilePath $LogFile -Append
