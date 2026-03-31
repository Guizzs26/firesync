# Configurações
$Container = "firesync_firebird"
$SQLSource = "./init-firebird.sql"
$DBPath    = "/firebird/data/pax.fdb"
$SQLDest   = "/tmp/init-firebird.sql"

Write-Host "--- Provisionamento Firebird 2.5 ---" -ForegroundColor Cyan

if (-not (Test-Path $SQLSource)) {
    Write-Host "Erro: Arquivo '$SQLSource' nao encontrado!" -ForegroundColor Red
    exit
}

$Status = docker inspect -f '{{.State.Running}}' $Container 2>$null
if ($Status -ne "true") {
    Write-Host "Erro: Container nao esta rodando." -ForegroundColor Red
    exit
}

Write-Host "Limpando resquicios..."
docker exec -u root $Container sh -c "rm -f $DBPath && chown -R firebird:firebird /firebird/data"

Write-Host "Copiando script SQL..."
docker cp $SQLSource "${Container}:${SQLDest}"

Write-Host "Executando ISQL..."
# Injetamos as variaveis de ambiente ISC_USER e ISC_PASSWORD diretamente no processo do exec
$Result = docker exec -i -e ISC_USER=SYSDBA -e ISC_PASSWORD=masterkey $Container /usr/local/firebird/bin/isql -q -i $SQLDest 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "Erro na execucao do SQL:" -ForegroundColor Red
    Write-Output $Result
} else {
    Write-Host "Banco e tabelas criados com sucesso!" -ForegroundColor Green
}

Write-Host "Validando existencia do arquivo .fdb..."
docker exec -i $Container ls -lh $DBPath