param(
    [Parameter(Mandatory=$true)][string]$Hash,
    [Parameter(Mandatory=$true)][string]$AudioId,
    [Parameter(Mandatory=$true)][string]$AlbumId,
    [string]$Name = 'test'
)
$ErrorActionPreference = 'Stop'
$body = @{
    appid = 1001; area_code = '1'; behavior = 'play'; clientver = '9020'
    need_hash_offset = 1; relate = 1; token = ''; userid = 2626431536; vip = 1
    resource = @(@{
        album_audio_id = $AudioId; album_id = $AlbumId; hash = $Hash
        id = 0; name = "$Name.mp3"; type = 'audio'
    })
} | ConvertTo-Json -Depth 6 -Compress
$h = @{ 'KG-RC' = '1'; 'KG-THash' = 'expand_search_manager.cpp:852736169:451'; 'User-Agent' = 'KuGou2012-9020-ExpandSearchManager' }
$resp = Invoke-WebRequest -Uri 'http://media.store.kugou.com/v1/get_res_privilege' -Method Post -Headers $h -Body $body -ContentType 'application/json; charset=utf-8' -UseBasicParsing
$j = $resp.Content | ConvertFrom-Json
$d = $j.data | Select-Object -First 1
Write-Output ("error_code={0} message={1}" -f $j.error_code, $j.message)
Write-Output ("quality={0} privilege={1} pay_type={2} price={3} old_hide={4} is_publish={5}" -f $d.quality, $d.privilege, $d.pay_type, $d.price, $d.old_hide, $d.is_publish)
Write-Output ("info keys: {0}" -f (($d.info | Get-Member -MemberType NoteProperty).Name -join ','))
Write-Output ("downloadurl = [{0}]" -f $d.info.downloadurl)
Write-Output ("image       = [{0}]" -f $d.info.image)
