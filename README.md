# NexusLLM — LLM Inference Gateway

> Production-grade LLM Inference Gateway with multi-model routing, Redis rate limiting, semantic caching, JWT + API key auth, Prometheus observability, and full AWS infrastructure via Terraform and Helm.

![Go](https://img.shields.io/badge/Go-1.22-00ADD8?style=flat&logo=go)
![Redis](https://img.shields.io/badge/Redis-7.0-DC382D?style=flat&logo=redis)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?style=flat&logo=postgresql)
![Prometheus](https://img.shields.io/badge/Prometheus-2.x-E6522C?style=flat&logo=prometheus)
![Terraform](https://img.shields.io/badge/Terraform-1.6-7B42BC?style=flat&logo=terraform)
![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28-326CE5?style=flat&logo=kubernetes)
![AWS](https://img.shields.io/badge/AWS-EKS%2FRDS%2FElastiCache-FF9900?style=flat&logo=amazonaws)
![License](https://img.shields.io/badge/License-MIT-green)

---

## Overview

NexusLLM is a self-hosted API gateway that sits in front of multiple LLM providers and handles the hard parts of production AI infrastructure:

- **Unified API** — one endpoint, multiple providers behind it
- **Smart routing** — automatic failover from OpenAI to Anthropic on error
- **Cost control** — per-tenant token budgets and usage tracking
- **Rate limiting** — Redis token bucket, configurable per API key tier
- **Semantic caching** — SHA-256 prompt hashing, eliminates duplicate LLM calls
- **Full observability** — Prometheus metrics + pre-built Grafana dashboards
- **Multi-tenant** — isolated quotas, costs, and audit logs per tenant

---

## Architecture

```
                         ┌─────────────────────────────────────────┐
                         │              NexusLLM Gateway            │
                         │                                          │
Client ─── HTTPS ──────► │  Auth MW ─► Rate Limiter ─► Cache Check │
                         │                    │              │      │
                         │                    │         Cache HIT   │
                         │                    ▼              │      │
                         │            LLM Router ◄───────────┘      │
                         │           /    |    \                     │
                         │      OpenAI Anthropic Gemini              │
                         │           \    |    /                     │
                         │            Fallback Chain                 │
                         │                    │                      │
                         │          Metrics + Audit Log              │
                         └─────────────────────────────────────────┘
                                    │                 │
                              PostgreSQL            Redis
                          (usage_logs,           (rate limits,
                           api_keys,              cache,
                           tenants)               sessions)
                                    │
                              Prometheus ──► Grafana
```

---

## Key Features

### Multi-Provider Routing with Automatic Failback
```
Request ──► OpenAI (primary)
              │
              └── Error / Unhealthy?
                    │
                    ▼
              Anthropic (fallback)
                    │
                    └── Error?
                          │
                          ▼
                    Gemini (fallback)
```

### Redis Token Bucket Rate Limiting
- Per-tenant, per-minute sliding window using Redis INCR + EXPIRE
- Returns `X-RateLimit-Limit` and `X-RateLimit-Remaining` headers
- Graceful fail-open if Redis is temporarily unavailable
- 429 responses with `Retry-After` header

### Semantic Cache
- SHA-256 hash of (messages + model) as cache key
- Configurable TTL (default 1 hour)
- `X-Cache: HIT/MISS` header on every response
- Cache hits tracked in Prometheus for hit-rate dashboards

### Dual Auth: API Keys + JWT
- **API keys**: `nxl_` prefixed, bcrypt hashed, prefix-indexed for O(1) lookup
- **JWT**: HS256, 15-minute expiry, role-based (admin / developer)
- Audit log: every key use recorded with timestamp in PostgreSQL

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22 |
| Auth | JWT (HS256) + bcrypt API keys |
| Rate Limiting | Redis token bucket |
| Caching | Redis + SHA-256 semantic hashing |
| Database | PostgreSQL 15 (multi-tenant schema) |
| Observability | Prometheus + Grafana |
| Containerization | Docker (multi-stage scratch image) |
| Orchestration | Kubernetes (EKS) + Helm |
| Infrastructure | Terraform (VPC, EKS, RDS, ElastiCache) |
| CI/CD | GitHub Actions (test → build → deploy) |
| LLM Providers | OpenAI, Anthropic, Gemini |

---

## Project Structure

```
NexusLLM/
├── cmd/gateway/           # Main entrypoint
├── internal/
│   ├── auth/              # JWT generation/validation, bcrypt API key store
│   ├── cache/             # SHA-256 semantic cache (Redis)
│   ├── config/            # Environment-based config
│   ├── db/                # PostgreSQL connection + schema migrations
│   ├── metrics/           # Prometheus counter/histogram/gauge definitions
│   ├── middleware/         # Auth and rate limiter HTTP middleware
│   ├── providers/         # OpenAI, Anthropic provider implementations
│   └── router/            # LLM router with fallback chain
├── terraform/
│   ├── main.tf            # VPC, subnets, security groups, module calls
│   ├── variables.tf       # Input variables with validation
│   ├── outputs.tf         # Cluster, RDS, Redis endpoints
│   └── modules/
│       ├── eks/           # EKS cluster + managed node group + IAM
│       ├── rds/           # PostgreSQL 15, Multi-AZ, encrypted, backups
│       └── elasticache/   # Redis 7.0 replication group, TLS
├── helm/nexusllm/
│   ├── Chart.yaml
│   ├── values.yaml        # Replicas, resources, autoscaling, ingress
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── hpa.yaml       # CPU + memory autoscaling (2-10 pods)
│       └── pdb.yaml       # PodDisruptionBudget (minAvailable: 1)
├── monitoring/
│   ├── prometheus.yml     # Scrape config
│   └── grafana/           # Auto-provisioned datasource
├── .github/workflows/
│   └── ci-cd.yml          # Test → Build → Deploy (staging + prod)
├── docker-compose.yml     # Full local stack
├── Dockerfile             # Multi-stage scratch image
└── Makefile               # Build, test, lint, docker, terraform shortcuts
```

---

## Quick Start (Local)

```bash
git clone https://github.com/MohammedZakirMemon/NexusLLM.git
cd NexusLLM

cp .env.example .env
# Set OPENAI_API_KEY or ANTHROPIC_API_KEY in .env

make docker-up
```

Services will be available at:

| Service | URL |
|---|---|
| Gateway API | http://localhost:8080 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin / admin) |

---

## API Usage

### Health Check
```bash
curl http://localhost:8080/health
# {"status":"ok","env":"development"}
```

### Chat Completion
```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer nxl_your_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Explain rate limiting in one sentence."}]
  }'
```

**Response:**
```json
{
  "id": "chatcmpl-abc123",
  "model": "gpt-4o-mini",
  "provider": "openai",
  "content": "Rate limiting controls the number of requests a client can make...",
  "prompt_tokens": 18,
  "completion_tokens": 42,
  "total_tokens": 60,
  "latency_ms": 843,
  "cost_usd": 0.0000000273
}
```

**Headers returned:**
```
X-Cache: MISS
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 59
```

Second identical request:
```
X-Cache: HIT       <- served from Redis, zero LLM cost
```

### Prometheus Metrics
```bash
curl http://localhost:8080/metrics | grep nexusllm
```

Key metrics:
```
nexusllm_requests_total{provider="openai",model="gpt-4o-mini",status="200"} 142
nexusllm_request_duration_ms_bucket{provider="openai",model="gpt-4o-mini",le="1000"} 138
nexusllm_tokens_total{provider="openai",model="gpt-4o-mini",type="prompt"} 2840
nexusllm_cost_usd_total{provider="openai",tenant="acme-corp"} 0.00042
nexusllm_cache_hits_total{type="hit"} 28
nexusllm_rate_limit_hits_total{tenant="free-tier",tier="free"} 3
nexusllm_active_requests 2
```

---

## Observability

### Prometheus Queries

**P99 latency by provider:**
```promql
histogram_quantile(0.99, rate(nexusllm_request_duration_ms_bucket[5m]))
```

**Cache hit rate:**
```promql
rate(nexusllm_cache_hits_total{type="hit"}[5m]) /
  rate(nexusllm_cache_hits_total[5m])
```

**Cost per provider (last hour):**
```promql
increase(nexusllm_cost_usd_total[1h])
```

**Rate limit rejection rate:**
```promql
rate(nexusllm_rate_limit_hits_total[1m])
```

---

## Deployment (AWS)

### 1. Provision Infrastructure
```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars
# Fill in db_username, db_password

terraform init
terraform plan
terraform apply
```

Provisions: VPC, public/private subnets, EKS cluster, RDS PostgreSQL (Multi-AZ), ElastiCache Redis (replication group), security groups with least-privilege ingress.

### 2. Deploy via Helm
```bash
aws eks update-kubeconfig --name nexusllm-prod --region us-east-1

helm upgrade --install nexusllm ./helm/nexusllm \
  --namespace nexusllm \
  --create-namespace \
  --set image.tag=sha-$(git rev-parse --short HEAD) \
  --set env.ENVIRONMENT=production \
  --wait
```

### 3. CI/CD (Automated)

GitHub Actions pipeline triggers automatically:

| Trigger | Pipeline |
|---|---|
| PR to `develop` or `main` | Test + Lint |
| Push to `develop` | Test → Build → Deploy to Staging |
| Push to `main` | Test → Build → Deploy to Production |

---

## Security

| Layer | Implementation |
|---|---|
| API Keys | bcrypt hashed, prefix-indexed, never stored raw |
| JWT | HS256, 15-minute expiry, role-based |
| Secrets | AWS Secrets Manager (production), .env (local) |
| Network | VPC private subnets, RDS/Redis never public |
| Encryption | RDS encrypted at rest, ElastiCache TLS in transit |
| IAM | Least-privilege roles per service (EKS, RDS, ECR) |
| Audit | Every API key use logged with timestamp + tenant |

---

## Performance

| Metric | Value |
|---|---|
| Cache hit response time | < 5ms |
| Rate limiter overhead | < 1ms (Redis pipeline) |
| Docker image size | ~10MB (scratch base) |
| HPA scaling range | 2 to 10 pods |
| RDS automated backups | 7-day retention |
| Redis replication | 2-node with automatic failover |

---

## Interview Talking Points

**Why Go?**
Go's concurrency model (goroutines, channels) handles high-throughput HTTP efficiently. The stdlib `net/http` is production-grade and the binary compiles to a single ~10MB scratch image.

**Why Redis for rate limiting instead of in-memory?**
In-memory rate limiting breaks across multiple pod replicas — each pod has its own counter. Redis gives a single consistent view across all instances, which is correct behavior in a distributed deployment.

**Why bcrypt prefix lookup for API keys?**
Full table scan + bcrypt compare on every request would be too slow. Storing a non-secret prefix (first 12 chars) lets us fetch only the candidate rows matching that prefix, then do bcrypt compare on the small result set.

**Why semantic caching at the gateway layer?**
Application code doesn't need to implement caching. The gateway intercepts identical requests transparently, returns cached responses with zero LLM cost, and reports cache hit rate in Prometheus.

**How does the fallback routing work?**
Each provider implements a `IsHealthy()` check. The router iterates the ordered provider list and calls the first healthy one. On error, it logs and tries the next. This gives automatic failover with no client-side changes needed.

---

## Local Development

```bash
# Build binary
make build

# Run tests with race detector
make test

# Run linter
make lint

# Start full stack
make docker-up

# View gateway logs
make docker-logs

# Tear down
make docker-down
```

---

## License

MIT — see [LICENSE](LICENSE)
