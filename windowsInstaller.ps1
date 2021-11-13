Write-Output "Creating any needed directories"

if (-not (Test-Path $HOME\bin)) {
    New-Item "~/bin" -ItemType Directory
}

if (-not (Test-Path $HOME\.packageless)) {
    New-Item $HOME\.packageless -ItemType Directory
    New-Item $HOME\.packageless\pims_config -ItemType Directory
    New-Item $HOME\.packageless\pims -ItemType Directory 
}

Write-Output "Downloading the executable"

Invoke-WebRequest https://github.com/everettraven/packageless/releases/latest/download/packageless-windows -OutFile $HOME\bin\packageless.exe

Write-Output "Adding packageless to PATH"

setx PATH "%PATH%;$HOME\bin\packageless"