SDK_PATH := $(shell xcrun --show-sdk-path)

build:
	CGO_ENABLED=1 \
	CGO_CXXFLAGS="-isysroot $(SDK_PATH) -I$(SDK_PATH)/usr/include/c++/v1" \
	go build -o slack-cli .

install: build
	cp slack-cli /usr/local/bin/

clean:
	rm -f slack-cli

.PHONY: build install clean
