param([string[]]$Keywords)
Add-Type -AssemblyName System.Web
$ErrorActionPreference = 'Stop'
$h = @{ 'KG-RC'='1'; 'KG-THash'='expand_search_manager.cpp:852736169:451'; 'User-Agent'='KuGou2012-9020-ExpandSearchManager' }
foreach ($kw in $Keywords) {
    $u = 'http://songsearch.kugou.com/song_search_v2?' + 'platform=AndroidFilter&iscorrection=1&keyword=' +
        [System.Web.HttpUtility]::UrlEncode($kw) + '&hifiquality=0&pagesize=50&PrivilegeFilter=0&page=1'
    $j = (Invoke-WebRequest -Uri $u -Headers $h -UseBasicParsing).Content | ConvertFrom-Json
    $free = @($j.data.lists | Where-Object { $_.Privilege -eq 0 -and $_.FileHash })
    Write-Output ("=== {0}: total={1} free(Privilege=0)={2}" -f $kw, $j.data.total, $free.Count)
    foreach ($it in $free | Select-Object -First 3) {
        Write-Output ("  Priv={0} SQPriv={1} audioid={2} album={3} size={4} hash={5}" -f $it.Privilege, $it.SQPrivilege, $it.Audioid, $it.AlbumID, $it.FileSize, $it.FileHash)
        Write-Output ("    {0} - {1}" -f $it.OriSongName, $it.SingerName)
    }
}
