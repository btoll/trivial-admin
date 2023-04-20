CC = go
CMD = run
TARGET = trivial-admin

.PHONY: build clean createCert

build:
	$(CC) build

clean:
	rm -rf $(TARGET)

createCert:
	go run createCert.go

