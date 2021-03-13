# Windows

## How to build

Build with Go 1.16 and MinGW.

```
# build syso
x86_64-w64-mingw32-windres -o main_windows_amd64.syso resource/shadow.rc

# use tags as shadow
go build -v -ldflags="-s -w -H=windowsgui" -trimpath -tags=""
```

