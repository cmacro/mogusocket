

.PHONY: build
buld:
	go build -o bin/server ./example/server/.
	go build -o bin/client ./example/client/.
	go build -o bin/autobahn ./example/autobahn/.
	