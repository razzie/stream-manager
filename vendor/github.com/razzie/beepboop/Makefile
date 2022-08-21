.DEFAULT_GOAL := beepboop
BUILDFLAGS := -mod=vendor -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR)

beepboop:
	go build $(BUILDFLAGS) .

fileserver:
	go build $(BUILDFLAGS) ./demo/fileserver

demo: fileserver

.PHONY: beepboop fileserver demo
