# terraform-dashboard

A scalable system for monitoring and analyzing Terraform infrastructure.

## terraform-indexer

A scalable, pluggable file ingestion and indexing service that:

- Pulls `.tfstate` and `.tf` files from external sources (e.g., S3, Kubernetes Secrets, Bitbucket)
- Parses the files into Terraform language constructs (resources, providers, variables, etc.)
- Stores the data in a searchable, normalized PostgreSQL database
- Runs as a long-lived process with Prometheus metrics and health checks

### Quick Start

**Using Docker Compose (Recommended):**

```bash
# Start the complete stack (PostgreSQL + terraform-indexer)
docker compose up

# Check health and metrics
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

**Building from source:**

```bash
cd terraform-indexer
go mod tidy
go build ./cmd/terraform-indexer

# Configure database connection in config.yaml
./terraform-indexer server --config config.yaml
```

### Architecture

```
terraform-dashboard/
├── terraform-indexer/           # Core indexing service
│   ├── cmd/
│   │   └── terraform-indexer/   # Main CLI entrypoint
│   ├── internal/
│   │   ├── collector/           # Collector plugins (S3, Kubernetes, Bitbucket)
│   │   ├── parser/              # tf / tfstate parsing
│   │   ├── writer/              # DB writers
│   │   ├── model/               # TerraformObject types
│   │   ├── queue/               # Parser and Writer queues
│   │   ├── db/                  # PostgreSQL client and schema
│   │   └── utils/               # Logging, metrics, utilities
│   ├── Dockerfile
│   └── config.yaml
├── docker-compose.yml
├── testdata/                    # Sample Terraform files
└── test.sh                      # Integration test script
```

### Components

- **Collectors**: Poll files from external sources (S3, Kubernetes, Bitbucket)
- **Parsers**: Parse `.tf` (HCL) and `.tfstate` (JSON) files into structured objects
- **Writers**: Store parsed objects in PostgreSQL with proper schema
- **Queues**: Manage processing pipeline between collectors, parsers, and writers
- **Metrics**: Prometheus metrics for monitoring and observability

### API Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics

### Database Schema

**terraform_files table:**
- Stores raw file content and metadata
- Indexed by source, file_type, content_hash

**terraform_objects table:**
- Stores parsed Terraform objects (resources, providers, variables, etc.)
- Indexed by type, resource_type, address, provider_name
- JSON configuration storage with full-text search capability

### Configuration

The service is configured via `config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  host: "localhost"
  port: 5432
  user: "terraform_indexer"
  password: "password"
  database: "terraform_dashboard"
  ssl_mode: "disable"

collectors:
  - name: "mock-s3"
    type: "mock"
    enabled: true
    settings:
      source: "s3"

polling:
  interval: "30s"

logging:
  level: "info"

metrics:
  enabled: true
  path: "/metrics"
```

### Development

Run tests:
```bash
./test.sh
```

Run locally with hot reload:
```bash
cd terraform-indexer
go run ./cmd/terraform-indexer server
```

### Metrics

Available Prometheus metrics:

- `terraform_indexer_files_collected_total` - Total files collected by source and type
- `terraform_indexer_objects_parsed_total` - Total objects parsed by type
- `terraform_indexer_objects_written_total` - Total objects written to database
- `terraform_indexer_processing_errors_total` - Processing errors by component
- `terraform_indexer_processing_duration_seconds` - Processing time histograms
- `terraform_indexer_queue_size` - Current queue sizes
- `terraform_indexer_worker_status` - Worker process status
