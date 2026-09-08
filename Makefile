STATICCHECK := honnef.co/go/tools/cmd/staticcheck@2025.1.1
WGO := github.com/bokwoon95/wgo@v0.7.1

.PHONY: web build check check-go check-web test run dev

web:                       ## build the frontend into internal/web/dist
	cd web && pnpm install --frozen-lockfile && pnpm build

build: web                 ## build the binary
	CGO_ENABLED=0 go build -trimpath -o bin/understudy ./cmd/understudy

check: check-go check-web  ## everything CI runs

check-go:                  ## gofmt, vet, staticcheck, tests
	test -z "$$(gofmt -l . | grep -v /dist/)" || { gofmt -l .; exit 1; }
	go vet ./...
	go run $(STATICCHECK) ./...
	go test ./...

check-web:                 ## prettier, eslint, svelte-check, build
	cd web && pnpm lint && pnpm check && pnpm build

test:
	go test ./...

run: build                 ## serve the editor locally
	./bin/understudy edit

dev:                       ## hot reload for both halves, see scripts/dev
	@WGO=$(WGO) scripts/dev
