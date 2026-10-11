.PHONY: run build check backend frontend

run: backend/conf.yaml frontend/conf.yaml build \
     frontend/node_modules/.package-lock.json
	node scripts/dev.mjs

frontend/node_modules/.package-lock.json: frontend/package.json \
                                         frontend/package-lock.json
	cd frontend && npm ci --ignore-scripts --no-audit --no-fund

backend/conf.yaml: backend/conf.example.yaml
	@test -f $@ || cp $< $@

frontend/conf.yaml: frontend/conf.example.yaml
	@test -f $@ || cp $< $@

build:
	cd backend && go build -trimpath -o bin/taskflow ./cmd/taskflow

backend: backend/conf.yaml
	cd backend && go run ./cmd/taskflow

frontend: frontend/conf.yaml frontend/node_modules/.package-lock.json
	cd frontend && npm run dev

check:
	cd framework && go vet ./... && go build ./...
	cd backend && go vet ./... && go build ./...
	cd examples/consumer && go vet ./... && go build -o /dev/null .
	@for file in frontend/src/*.js frontend/public/*.js frontend/*.mjs; do \
		node --check "$$file" || exit 1; \
	done
	node --check scripts/dev.mjs
	bash -n scripts/deploy.sh
	bash -n scripts/install-deps.sh
