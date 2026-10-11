param([Parameter(Mandatory=$true)][string]$Keyword)
$ErrorActionPreference = 'Stop'
$pairs = @(
    'platform=AndroidFilter', 'iscorrection=1',
    'keyword=' + [System.Web.HttpUtility]::UrlEncode($Keyword),
    'hifiquality=0', 'pagesize=8', 'PrivilegeFilter=0', 'page=1'
) -join '&'
$h = @{ 'KG-RC' = '1'; 'KG-THash' = 'expand_search_manager.cpp:852736169:451'; 'User-Agent' = 'KuGou2012-9020-ExpandSearchManager' }
$resp = Invoke-WebRequest -Uri ("http://songsearch.kugou.com/song_search_v2?$q") -Headers $h -UseBasicParsing
$j = $resp.Content | ConvertFrom-Json
foreach ($it in $j.data.lists) {
    $name = if ($it.Suffix) { "$($it.OriSongName) $($it.Suffix)" } else { $it.OriSongName }
    Write-Output ("Priv={0,-3} Audioid={1,-10} Album={2,-10} Size={3,-11} Hash={4}" -f $it.Privilege, $it.Audioid, $it.AlbumID, $it.FileSize, $it.FileHash)
    Write-Output ("    {0} - {1}" -f $name, ($it.Singers | ForEach-Object { $_.name } -join '/'))
}
