.PHONY: all clean

all: bin/buffbot-32.exe bin/buffbot-64.exe

bin/rsrc:
	go build -o bin/rsrc github.com/akavel/rsrc

cmd/buffbot/rsrc.syso: bin/rsrc build/buffbot.manifest
	bin/rsrc -manifest build/buffbot.manifest -o cmd/buffbot/rsrc.syso

bin/buffbot-32.exe:  cmd/buffbot/rsrc.syso $(shell find cmd/buffbot -name *.go) $(shell find internal -name *.go)
	GOARCH=386 go build -o bin/buffbot-32.exe -ldflags="-H windowsgui" github.com/GontikR99/buffbot/cmd/buffbot

bin/buffbot-64.exe:  cmd/buffbot/rsrc.syso $(shell find cmd/buffbot -name *.go) $(shell find internal -name *.go)
	GOARCH=amd64 go build -o bin/buffbot-64.exe -ldflags="-H windowsgui" github.com/GontikR99/buffbot/cmd/buffbot

clean:
	rm -f bin/* cmd/buffbot/rsrc.syso
