.PHONY: build run test lint docker-up docker-down clean

build:
	go build -ldflags="-s -w" -o bin/nexusllm ./cmd/gateway

run:
	go run ./cmd/gateway

test:
	go test ./... -v -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

docker-logs:
	docker compose logs -f gateway

clean:
	rm -rf bin/ coverage.out coverage.html

# Terraform shortcuts
tf-init:
	cd terraform && terraform init

tf-plan:
	cd terraform && terraform plan

tf-apply:
	cd terraform && terraform apply -auto-approve

tf-destroy:
	cd terraform && terraform destroy -auto-approve
