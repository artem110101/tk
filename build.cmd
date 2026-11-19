@echo off
setlocal
pushd "%~dp0" || exit /b 1
set "CGO_ENABLED=1"
go test ./... && go build -trimpath -o bin\tk.exe .
set "rc=%ERRORLEVEL%"
popd
exit /b %rc%
