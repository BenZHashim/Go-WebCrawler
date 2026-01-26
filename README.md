# Go-WebCrawler

A high-performance, concurrent web crawler built in Go. This project features a **hybrid scraping engine** that intelligently switches between static HTML fetching and headless Chrome (Chromedp) to handle dynamic JavaScript-heavy websites.

Designed for scalability and extensibility, it uses a worker pool architecture to process pages in parallel while respecting domain-specific rate limits and `robots.txt` rules.

## 🚀 Key Features

* **Hybrid Parsing Engine**: Starts with fast static analysis (`net/http`) and automatically upgrades to a headless browser (`chromedp`) if it detects challenges like "Enable JavaScript" warnings or empty content.
* **Concurrency**: Implements a worker-pool pattern to crawl multiple pages simultaneously, controlled by configurable batch sizes and worker counts.
* **Politeness & Compliance**: Built-in `robots.txt` parsing and per-domain rate limiting to ensure ethical crawling.
* **Modular Architecture**: Interface-based design (`Processor` and `Sink`) makes it easy to swap out parsing logic or storage backends.
* **Stealth Mode**: Includes browser fingerprinting mitigations (User-Agent rotation, stealth scripts, and human-like jitter) to avoid detection.
* **Persistent Storage**: Automatically saves crawled content and page metadata to a PostgreSQL database.
* **Docker Ready**: Fully containerized with Docker and Docker Compose for easy deployment.
## 🧠 How It Works

1.  **Initialization**: The `Engine` starts with a seed URL and initializes the `DomainManager` and `Storage`.
2.  **Dispatch**: The `Engine` distributes URLs to a pool of workers.
3.  **Processing**:
    * The `Processor` checks if the URL requires dynamic fetching.
    * It parses the page to extract text content, metadata, and outbound links.
    * If the page is "suspicious" (e.g., Cloudflare challenge), it marks the domain as dynamic and retries with a headless browser.
4.  **Filtering**: Outbound links are filtered to ensure they stay within the target domain or match specific product patterns.
5.  **Storage**: Extracted data is sent to a `Sink`, which batches records and efficiently inserts them into PostgreSQL.

## 🏗 Architecture

```mermaid
graph TD
    %% Nodes representing external systems
    User((User))
    Web[Target Website]
    DB[(PostgreSQL)]
    Chrome[Headless Chrome]

    %% Main System Components
    subgraph "Go Crawler Engine"
        Orchestrator[Engine Orchestrator]
        WorkList[WorkList Channel]
        Worker[Worker Pool]
        
        
        subgraph "Processing Pipeline"
            DM[Domain Manager]
            Proc[Page Processor]
            Parser[Hybrid Parser]
            Results[Results Channel]
            Sink[Storage Sink]
        end
    end

    %% Data Flow Connections
    User -- "StartURL" --> Orchestrator
    Orchestrator -- "Seed" --> WorkList
    WorkList -- "Pop URL" --> Worker
    
    Worker -- "Rate Limit / Robots.txt" --> DM
    Worker -- "Process(URL)" --> Proc
    
    Proc -- "Fetch Content" --> Parser
    Parser -- "HTTP Get" --> Web
    Parser -- "Fallback / Dynamic" --> Chrome
    Chrome -- "Render" --> Web
    
    Proc -- "Extracted PageData" --> Results
    Proc -- "Discovered Links" --> WorkList
    
    Results -- "Buffer & Batch" --> Sink
    Sink -- "INSERT" --> DB
```

## 🧩 Code Architecture

While "How It Works" describes the data flow, this diagram shows how the Go interfaces (`Processor`, `Sink`) decouple the crawling logic from the storage and parsing implementations.

```mermaid
classDiagram
    direction TB

    class Engine {
        +Run(seeds)
        -startCrawlWorker()
        -startStorageWorker()
    }

    class Processor {
        <<interface>>
        +Process(url) (Data, Links, error)
    }

    class Sink {
        <<interface>>
        +Save(batch) error
    }

    class Parser {
        +FetchStatic()
        +FetchDynamic()
        +Parse()
    }

    class DomainManager {
        +Wait(url)
        +IsAllowed(url)
    }

    %% Relationships
    Engine --> Processor : Uses
    Engine --> Sink : Uses
    Engine --> DomainManager : Enforces Rules
    Processor --> Parser : Delegates Fetching
    
    %% Implementations
    class PageProcessor {
        +Filter URLFilter
    }
    class PageSink {
        +db *sql.DB
    }
    
    PageProcessor ..|> Processor : Implements
    PageSink ..|> Sink : Implements
```
## 🛠 Tech Stack

* **Language**: Go 1.24
* **Browser Automation**: [Chromedp](https://github.com/chromedp/chromedp)
* **Database**: PostgreSQL
* **Migrations**: SQL-based schema initialization
* **Containerization**: Docker & Docker Compose

## 📦 Project Structure

    ├── cmd/crawler/         # Entry point (main.go)
    ├── internal/
    │   ├── config/          # Configuration management via env vars
    │   ├── crawler/         # Core crawling logic (Engine, Parser, Filters)
    │   └── storage/         # Database persistence (Sinks)
    ├── migrations/          # SQL scripts for database initialization
    ├── pkg/models/          # Shared data structures (PageData, URLQueue)
    └── Dockerfile           # Multi-stage build for the crawler

## ⚙️ Configuration

The application is configured via environment variables or a `.env` file. The following variables are available:

| Variable      | Default | Description |
| :---          | :---    | :---        |
| `DB_URL`      | *Required* | Connection string for PostgreSQL |
| `START_URL`   | `https://www.hollywoodreporter.com` | The initial URL to start crawling |
| `WORKERS`     | `10`    | Number of concurrent crawling workers |
| `BATCH_SIZE`  | `20`    | Number of items to buffer before writing to DB |
| `RATE_LIMIT`  | `2s`    | Minimum delay between requests to the same domain |

## 🏁 Getting Started

### Prerequisites

* Go 1.24+
* Docker & Docker Compose (optional, for containerized run)
* PostgreSQL (if running locally without Docker)

### Option 1: Running with Docker (Recommended)

This method spins up both the crawler and the PostgreSQL database automatically.

**On Mac / Linux** (using Make):

    # Start the crawler and database
    make docker-up

    # View logs
    make docker-logs

    # Stop all containers
    make docker-down

**On Windows** (using PowerShell or CMD):

    # Start the crawler and database
    docker-compose up --build

    # View logs
    docker-compose logs -f

    # Stop all containers
    docker-compose down

### Option 2: Running Locally

If you prefer to run the Go binary directly on your machine:

1.  **Set up the database**:
    Ensure you have a PostgreSQL instance running. You can use the schema in `migrations/init.sql` to create the necessary tables.

2.  **Configure the environment**:
    Create a `.env` file in the root directory:

        DB_URL=postgres://user:pass@localhost:5432/crawler_db?sslmode=disable
        WORKERS=5
        START_URL=https://example.com

3.  **Run the application**:

    **On Mac / Linux**:

        make run

    **On Windows**:

        go run cmd/crawler/main.go

## 🤝 Contributing

Contributions are welcome! Please ensure any new logic includes tests and respects the existing `internal` package structure.

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request


## 📄 License
[MIT License](LICENSE)