# ABT Corporation Analytics Dashboard

A **production-ready**, enterprise-grade business analytics dashboard built with Go, featuring comprehensive security middleware, graceful shutdown, structured error handling, and real-time data visualization.

## 🏗️ Technical Stack

### Core Technologies
- **Backend**: Go 1.24+ with high-performance analytics engine
- **Templates**: Templ for type-safe HTML rendering
- **Frontend**: DataStar v1 + Chart.js 4.4.7 for reactive UI
- **Data Processing**: Concurrent CSV parsing with error handling and recovery

### Production Infrastructure
- **Configuration**: Environment-based config with validation
- **Security**: Rate limiting, CORS, CSRF protection, security headers
- **Observability**: Structured JSON logging and distributed tracing
- **Error Handling**: Structured responses with request IDs and proper HTTP codes
- **Graceful Shutdown**: Context-aware shutdown with configurable timeouts

## 🏗️ Project Structure

```
qa-sse/
├── cmd/web/                    # Application entry point
│   ├── main.go                 # Production-ready main with all middleware
│   └── main_test.go           # Integration tests for HTTP endpoints
├── internal/
│   ├── config/                # Configuration management
│   │   └── config.go          # Environment-based configuration
│   ├── errors/                # Structured error handling
│   │   └── errors.go          # Error types and response formatting
│   ├── handlers/              # HTTP request handlers
│   │   ├── api.go             # REST API with error handling & tracing
│   │   ├── api_test.go        # Comprehensive handler tests
│   │   ├── sse.go             # Server-Sent Events handlers
│   │   └── sse_test.go        # SSE endpoint tests
│   ├── middleware/            # Security and observability middleware
│   │   └── middleware.go      # CORS, rate limiting, tracing, logging
│   ├── models/                # Data models and structures
│   │   └── transaction.go     # Transaction and analytics models
│   ├── observability/         # Logging and tracing
│   │   ├── logger.go          # Structured logging configuration
│   │   └── tracing.go         # Distributed tracing implementation
│   ├── server/                # HTTP server and graceful shutdown
│   │   ├── server.go          # Server setup and routing
│   │   └── shutdown.go        # Graceful shutdown with hooks
│   ├── services/              # Business logic services
│   │   ├── analytics.go       # High-performance analytics engine
│   │   └── analytics_test.go  # Service unit tests
│   └── ui/templates/          # Frontend templates
│       ├── base.templ         # Base template layout
│       ├── dashboard.templ    # Main dashboard template
│       └── templates_utils.go # Template utility functions
├── data.csv                   # Sample transaction data
├── Taskfile.yml              # Task runner configuration
├── .env.example              # Production configuration template
└── go.mod                    # Go module with production dependencies
```

## 🔧 Quick Start

### Prerequisites

**Data Setup**:
```bash
# Extract the provided dataset and place it in the root directory
unzip GO_test_5m.zip
mv extracted_data.csv data.csv  # Rename to data.csv and place in root directory
```

Install required tools:
```bash
# Install Go Task runner
go install github.com/go-task/task/v3/cmd/task@latest

# Install templ for template generation
go install github.com/a-h/templ/cmd/templ@latest

# Install Air for live reloading (optional, for development)
go install github.com/cosmtrek/air@latest
```

### Running the Application

```bash
# Install dependencies
task deps

# Generate templates first
task build:templ

# Build and run (production)
task build
./bin/main

# Or run directly with template generation
task run

# Development with live reload
task dev
```

Dashboard available at: **http://localhost:8084**

### Production Configuration

```bash
# Copy example configuration
cp .env.example .env

# Edit configuration for your environment
vim .env

# Run with production settings
./bin/main
```

### Available Tasks

Run `task --list` to see all available commands:

- `task deps` - Install dependencies and download modules
- `task build:templ` - Generate templates from .templ files
- `task build` - Build the application to bin/main
- `task run` - Run the application with template generation
- `task dev` - Start development with live reload
- `task test` - Run comprehensive test suite
- `task test:cover` - Generate test coverage report
- `task clean` - Clean build artifacts and generated files
- `task fmt` - Format Go code and Templ templates
- `task vet` - Run Go static analysis
- `task check` - Run format, vet, and test (full validation)

## 📡 API Endpoints

### REST API Endpoints
All endpoints return structured JSON responses with success/error wrappers:

```json
{
  "success": true,
  "data": { /* your data */ }
}
```

| Endpoint | Method | Description | Cache | Security |
|----------|--------|-------------|--------|----------|
| `GET /` | GET | Main dashboard interface | 5min | CSRF Protected |
| `GET /health` | GET | Health check endpoint | No cache | Public |
| `GET /admin/stats` | GET | System statistics | No cache | Protected |
| `GET /api/country-revenue` | GET | Country revenue data | 5min | Rate Limited |
| `GET /api/top-products` | GET | Top 20 products by frequency | 5min | Rate Limited |
| `GET /api/monthly-sales` | GET | Monthly sales volume | 5min | Rate Limited |
| `GET /api/top-regions` | GET | Top 30 regions by revenue | 5min | Rate Limited |

### Server-Sent Events (SSE) Endpoints
| Endpoint | Method | Description | Response Format |
|----------|--------|-------------|-----------------|
| `GET /sse/country-revenue` | GET | Real-time country table updates | SSE HTML |
| `GET /sse/top-products` | GET | Real-time product chart data | SSE JSON |
| `GET /sse/monthly-sales` | GET | Real-time monthly chart data | SSE JSON |
| `GET /sse/top-regions` | GET | Real-time region chart data | SSE JSON |

### Error Responses

```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests",
    "timestamp": "2024-01-01T12:00:00Z",
    "request_id": "abc123-def456"
  }
}
```

## 📊 Observability

### Structured Logging
```json
{
  "time": "2024-01-01T12:00:00Z",
  "level": "INFO",
  "msg": "request completed",
  "method": "GET",
  "path": "/api/country-revenue",
  "status": 200,
  "duration": "5.2ms",
  "request_id": "abc123-def456"
}
```

### Distributed Tracing
- Request-level tracing with span IDs
- Operation timing and error tracking
- Context propagation across service calls

## 📄 Data Format

Place your `data.csv` file with this structure:

```csv
transaction_id,transaction_date,user_id,country,region,product_id,product_name,category,price,quantity,total_price,added_date,stock_quantity
T001,2023-01-15,U001,USA,California,P001,Laptop,Electronics,999.99,1,999.99,2023-01-01,50
```

The application supports flexible CSV formats and handles various column arrangements with error recovery.

## 🧪 Testing

```bash
# Run all tests
task test

# Generate coverage report  
task test:cover

# Run tests with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./...
```

## 🏛️ Architecture

### Core Components

1. **Configuration Management** (`internal/config/`)
   - Environment-based configuration with validation
   - Type-safe configuration parsing
   - Development and production profiles

2. **Security Middleware** (`internal/middleware/`)
   - Rate limiting with IP-based buckets
   - CORS with configurable origins
   - Security headers and CSRF protection
   - Request ID generation and propagation

3. **Observability Stack** (`internal/observability/`)
   - Structured JSON logging with levels
   - Distributed tracing with span context
   - Request correlation and timing

4. **Error Handling** (`internal/errors/`)
   - Structured error types with codes
   - HTTP status code mapping
   - Request ID correlation for debugging

5. **Analytics Engine** (`internal/services/analytics.go`)
   - Concurrent CSV processing with worker pools
   - In-memory caching with binary GOB serialization
   - Precomputed aggregations for O(1) query performance
   - Thread-safe operations with read-write mutexes

6. **HTTP Server** (`internal/server/`)
   - RESTful API with comprehensive middleware
   - Server-Sent Events for real-time updates
   - Graceful shutdown with configurable timeouts
   - Context-aware request handling

### Data Flow

1. **Configuration Loading**: Environment variables → Validated config struct
2. **Middleware Chain**: Security → Logging → Tracing → Business logic
3. **CSV Processing**: Worker pools → Error handling → In-memory cache
4. **API Layer**: Structured responses → Error handling
5. **Frontend**: Reactive UI → SSE streaming → Chart.js visualizations

## 🚀 Production Deployment

### Quick Production Setup

```bash
# 1. Configure environment
cp .env.example .env
vim .env  # Configure for production

# 2. Build optimized binary
task build

# 3. Run with production settings
SERVER_HOST=0.0.0.0 \
SERVER_PORT=8084 \
LOG_LEVEL=info \
LOG_FORMAT=json \
SECURITY_RATE_LIMIT_ENABLED=true \
./bin/main
```

### Environment Variables

Essential production settings:

```bash
# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8084
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
SERVER_SHUTDOWN_TIMEOUT=30s

# Security
SECURITY_CSRF_ENABLED=true
SECURITY_RATE_LIMIT_ENABLED=true
SECURITY_RATE_LIMIT_RPS=100
SECURITY_ALLOWED_ORIGINS=https://givendomain.com

# Observability
LOG_LEVEL=info
LOG_FORMAT=json

# Data
CSV_FILE=production-data.csv
```

## 📦 Dependencies

### Core Production Dependencies
- `github.com/a-h/templ v0.3.943` - Type-safe HTML templates
- `github.com/starfederation/datastar-go v1.0.2` - SSE/reactive framework
- `golang.org/x/time v0.12.0` - Rate limiting
- `golang.org/x/sync v0.16.0` - Extended sync primitives

### Development Tools
- **Templ CLI**: Template generation from .templ files
- **Air**: Live reloading during development
- **Task**: Build automation and task management






