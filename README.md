# Windows

## How to build

Build with Go 1.16 and MinGW.

```
# build syso
x86_64-w64-mingw32-windres -o main_windows_amd64.syso resource/shadow.rc

# use tags as shadow
go build -v -ldflags="-s -w -H=windowsgui" -trimpath -tags=""
```
## How to use
Put `config.json` in the directory of `shadow-windows.exe`. More information about `config.json`, please click [this](https://github.com/imgk/shadow/tree/main/doc).

