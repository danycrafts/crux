.PHONY: build build-daemon build-cli build-dashboard install test clean

VERSION ?= 0.1.0
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build: build-daemon build-cli build-dashboard

build-daemon:
	cd services/daemon && go build $(LDFLAGS) -o ../../bin/cruxd ./cmd/cruxd

build-cli:
	cd services/cli && go build $(LDFLAGS) -o ../../bin/crux ./cmd/crux

build-dashboard:
	cd services/dashboard && go build $(LDFLAGS) -o ../../bin/crux-dashboard ./cmd/dashboard

install: build
	mkdir -p $(GOPATH)/bin /usr/local/bin 2>/dev/null || true
	cp bin/crux bin/cruxd bin/crux-dashboard /usr/local/bin/ 2>/dev/null || cp bin/crux bin/cruxd bin/crux-dashboard $(GOPATH)/bin/

test:
	cd services/daemon && go test ./...
	cd services/cli && go test ./...
	cd services/dashboard && go test ./...

clean:
	rm -rf bin/

# Cross-compilation targets
release-linux-amd64:
	cd services/daemon && GOOS=linux GOARCH=amd64 go build -o ../../dist/cruxd_linux_amd64 ./cmd/cruxd
	cd services/cli && GOOS=linux GOARCH=amd64 go build -o ../../dist/crux_linux_amd64 ./cmd/crux
	cd services/dashboard && GOOS=linux GOARCH=amd64 go build -o ../../dist/crux-dashboard_linux_amd64 ./cmd/dashboard

release-darwin-amd64:
	cd services/daemon && GOOS=darwin GOARCH=amd64 go build -o ../../dist/cruxd_darwin_amd64 ./cmd/cruxd
	cd services/cli && GOOS=darwin GOARCH=amd64 go build -o ../../dist/crux_darwin_amd64 ./cmd/crux
	cd services/dashboard && GOOS=darwin GOARCH=amd64 go build -o ../../dist/crux-dashboard_darwin_amd64 ./cmd/dashboard

release-darwin-arm64:
	cd services/daemon && GOOS=darwin GOARCH=arm64 go build -o ../../dist/cruxd_darwin_arm64 ./cmd/cruxd
	cd services/cli && GOOS=darwin GOARCH=arm64 go build -o ../../dist/crux_darwin_arm64 ./cmd/crux
	cd services/dashboard && GOOS=darwin GOARCH=arm64 go build -o ../../dist/crux-dashboard_darwin_arm64 ./cmd/dashboard

release-windows-amd64:
	cd services/daemon && GOOS=windows GOARCH=amd64 go build -o ../../dist/cruxd_windows_amd64.exe ./cmd/cruxd
	cd services/cli && GOOS=windows GOARCH=amd64 go build -o ../../dist/crux_windows_amd64.exe ./cmd/crux
	cd services/dashboard && GOOS=windows GOARCH=amd64 go build -o ../../dist/crux-dashboard_windows_amd64.exe ./cmd/dashboard
