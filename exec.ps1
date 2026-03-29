$Containers = @("firesync_firebird")
$SQLSource = "./init-firebird.sql"
$DBPath = "/firebird/data/pax.fdb"
$SQLDest = "/tmp/init-firebird.sql"

Write-Host "--- Starting Firebird provisioning ---" -ForegroundColor Cyan

if (-not (Test-Path $SQLSource)) {
    Write-Host "Error: $SQLSource not found!" -ForegroundColor Red
    exit
}

foreach ($Container in $Containers) {
    Write-Host ""
    Write-Host ">>> Unit: $Container <<<" -ForegroundColor Yellow

    $Status = docker inspect -f '{{.State.Running}}' $Container 2>$null
    if ($Status -ne "true") {
        Write-Host "Container not running. Skipping..." -ForegroundColor Red
        continue
    }

    Write-Host "Cleaning old database..."
    docker exec -i $Container rm -f $DBPath

    Write-Host "Copying SQL..."
    docker cp $SQLSource "${Container}:${SQLDest}"

    Write-Host "Creating database..."
    docker exec -i $Container sh -c "/usr/local/firebird/bin/isql -i $SQLDest"

    Write-Host "Validating..."
    docker exec -i $Container ls -lh $DBPath
}