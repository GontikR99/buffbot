.PHONY: all clean

all: bin/buffbot2.exe

bin/rsrc:
	go build -o bin/rsrc github.com/akavel/rsrc

cmd/buffbot/rsrc.syso: bin/rsrc build/buffbot.manifest
	bin/rsrc -manifest build/buffbot.manifest -o cmd/buffbot/rsrc.syso

bin/buffbot2.exe:  cmd/buffbot/rsrc.syso $(shell find cmd/buffbot -name *.go) $(shell find internal -name *.go)
	go build -o bin/buffbot2.exe -ldflags="-H windowsgui" github.com/GontikR99/buffbot/cmd/buffbot

clean:
	rm -f bin/* cmd/buffbot/rsrc.syso
