$authorizedKeys = "C:\Users\ogc\.ssh\authorized_keys"
$githubUsers = "C:\Users\ogc\e2e-testing\.ci\ansible\github-ssh-keys" 

$stream_reader1 = [System.IO.StreamReader]::new($githubUsers)
while (($githubUser =$stream_reader1.ReadLine()) -ne $null) {
  $source = "https://github.com/${githubUser}.keys"
  $destination = "C:\Users\ogc\${githubUser}.keys"

  Invoke-WebRequest -Uri $source -OutFile $destination 
  Get-Content "$destination" | ForEach-Object { 
    if($_ -match $regex){
      Write-Output $line 
    }
  }

  $stream_reader2 = [System.IO.StreamReader]::new($destination) 
  while (($current_key =$stream_reader2.ReadLine()) -ne $null)
  {
    Add-Content -Path $authorizedKeys -Value "$current_key"
  }

  $stream_reader2.Close()
  $stream_reader2.Dispose()
  Remove-Item $destination -Force
}

$stream_reader1.Close()
$stream_reader1.Dispose()
