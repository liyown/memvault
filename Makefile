.PHONY: build test install clean

build:
	cd web && npm ci --no-audit --no-fund && npm run build
	go build -trimpath -o bin/memvault ./cmd/memvault

test:
	go test -race ./... -count=1 -timeout=120s
	go vet ./...
	cd web && npx tsc --noEmit

install: build
	mkdir -p "$(HOME)/.local/bin"
	@test ! -f "$(HOME)/.local/bin/memvault" || cp "$(HOME)/.local/bin/memvault" "$(HOME)/.local/bin/memvault.previous"
	cp bin/memvault "$(HOME)/.local/bin/memvault.installing"
	chmod 755 "$(HOME)/.local/bin/memvault.installing"
	mv "$(HOME)/.local/bin/memvault.installing" "$(HOME)/.local/bin/memvault"

clean:
	rm -rf bin
