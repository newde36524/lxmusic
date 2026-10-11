param([string]$Project = 'C:\lx-rl\lxmusic')

# 'go tool dist list' prints pairs slash-separated (aix/ppc64), not whitespace.
$pairs = (go tool dist list) | ForEach-Object {
  $p = $_.Trim() -split '/', 2
  if ($p.Count -eq 2 -and $p[0] -ne '' -and $p[1] -ne '') {
    [pscustomobject]@{ GOOS = $p[0]; GOARCH = $p[1] }
  }
}

$pass = 0
$fail = [System.Collections.Generic.List[string]]::new()
$err = [System.Collections.Generic.List[string]]::new()

foreach ($t in $pairs) {
  Set-Item Env:GOOS   -Value $t.GOOS
  Set-Item Env:GOARCH -Value $t.GOARCH
  Set-Item Env:CGO_ENABLED -Value '0'

  $out = go build -C $Project -o (Join-Path $env:TEMP "lxm_$($t.GOOS)_$($t.GOARCH)") 2>&1
  if ($LASTEXITCODE -eq 0) {
    $pass++
  } else {
    $fail += "$($t.GOOS)/$($t.GOARCH)"
    $first = ($out | Out-String).Trim() -split "`n" | Select-Object -First 2
    $err += ("{0}/{1} : {2}" -f $t.GOOS, $t.GOARCH, ($first -join ' | '))
  }
}

Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue

Write-Host ("PASS: {0} / {1}" -f $pass, $pairs.Count)
Write-Host ""
Write-Host ("FAIL ({0})" -f $fail.Count)
foreach ($f in $err) { Write-Host "  $f" }
