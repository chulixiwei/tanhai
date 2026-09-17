@echo off
REM 探海 (Tanhai) 跨平台编译脚本 (Windows 版)
REM 生成 Windows / Linux / macOS 多平台二进制

setlocal enabledelayedexpansion

set VERSION=1.0.0
set APP_NAME=tanhai
set LDFLAGS=-s -w

echo [*] 探海 v%VERSION% 跨平台编译
echo.

if not exist dist mkdir dist

REM Windows amd64
echo [*] 编译 windows/amd64 -^> dist\tanhai.exe
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags="%LDFLAGS%" -o dist\tanhai.exe main.go

REM Linux amd64
echo [*] 编译 linux/amd64 -^> dist\tanhai-linux
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags="%LDFLAGS%" -o dist\tanhai-linux main.go

REM macOS amd64
echo [*] 编译 darwin/amd64 -^> dist\tanhai-darwin-amd64
set GOOS=darwin
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags="%LDFLAGS%" -o dist\tanhai-darwin-amd64 main.go

REM macOS arm64 (Apple Silicon)
echo [*] 编译 darwin/arm64 -^> dist\tanhai-darwin-arm64
set GOOS=darwin
set GOARCH=arm64
set CGO_ENABLED=0
go build -trimpath -ldflags="%LDFLAGS%" -o dist\tanhai-darwin-arm64 main.go

echo.
echo [*] 编译完成，产物在 dist\ 目录
dir dist

endlocal