param(
    [Parameter(Mandatory=$true)][string]$File,
    [Parameter(Mandatory=$true)][string]$Needle,
    [int]$Before = 300,
    [int]$After = 900,
    [int]$Max = 2
)
$sr = New-Object System.IO.StreamReader($File)
try {
    $buf = New-Object char[] 65536
    $pos = 0
    $seen = 0
    while ($true -and $seen -lt $Max) {
        $read = $sr.Read($buf, 0, $buf.Length)
        if ($read -le 0) { break }
        $chunk = New-Object System.Text.StringBuilder($read)
        [void]$chunk.Append($buf, 0, $read)
        $text = $chunk.ToString()
        $start = 0
        while ($true) {
            $i = $text.IndexOf($Needle, $start, [System.StringComparison]::Ordinal)
            if ($i -lt 0) { break }
            $lo = $i - $Before
            if ($lo -lt 0) { $lo = 0 }
            $hi = $i + $Needle.Length + $After
            if ($hi -gt $text.Length) { $hi = $text.Length }
            Write-Output ("=== hit @ absolute " + ($pos + $i) + " ===")
            Write-Output ($text.Substring($lo, $hi - $lo))
            Write-Output ""
            $seen++
            $start = $i + $Needle.Length
            if ($seen -ge $Max) { break }
        }
        $pos += $read
    }
}
finally { $sr.Dispose() }
