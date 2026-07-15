# Package Implementation TODO

> Consolidated list of packages needed to fully support the 120 services.

---

## Legend
- ✅ = Exists
- 🔄 = Partially exists
- ❌ = Missing

---

## What Already Exists (Summary)

| Domain | Existing Packages |
|--------|-------------------|
| **Rate Limiting** | `pkg/algorithms/ratelimit/*`, `pkg/api/ratelimit/*` |
| **Sharding** | `pkg/database/sharding/*`, `pkg/database/partitioning/*` |
| **Distributed Lock** | `pkg/concurrency/distlock/*` |
| **Vector Search** | `pkg/database/vector/*`, `pkg/database/rerank/*` |
| **Big Data** | `pkg/data/bigdata/*` (MapReduce, Spark, Parquet, Avro, DuckDB) |
| **Auth** | `pkg/auth/*` (JWT, OAuth2 AS memory, OIDC verify/exchange, MFA, Social) |
| **Messaging** | `pkg/messaging/*` (Kafka, NATS, RabbitMQ, SQS, SNS, Pub/Sub) |
| **Cache** | `pkg/cache/*` (Redis, memory) |
| **Blob** | `pkg/blob/*` (S3, GCS, Azure) |
| **Resilience** | `pkg/resilience/*` (Circuit breaker, retry, timeout, bulkhead) |

---

## 1. AI & Machine Learning (`pkg/ai`)

> **Path note:** There is no separate `pkg/ai/llm` tree. LLM APIs live under
> `pkg/ai/genai/llm` (and embeddings under `pkg/ai/nlp/embedding`). Rows below that
> still say `pkg/ai/llm/...` are a historical ledger alias — treat them as pointing
> at the corresponding `pkg/ai/genai/llm/...` packages. Do not create a dual tree.

### LLM Core (`pkg/ai/genai/llm` — formerly listed as `pkg/ai/llm`)
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/ai/genai/llm` | ✅ | llm-gateway / llm-core | LLM Client (`Chat` + `StreamChat`) |
| `pkg/ai/genai/llm/adapters/openai` | ✅ | llm-gateway | OpenAI Adapter |
| `pkg/ai/genai/llm/adapters/anthropic` | ✅ | llm-gateway | Anthropic Adapter |
| `pkg/ai/genai/llm/adapters/gemini` | ✅ | llm-gateway | Google Gemini Adapter |
| `pkg/ai/genai/llm/adapters/ollama` | ✅ | llm-gateway | Ollama Adapter (Local LLM) |
| `pkg/ai/genai/llm/adapters/memory` | ✅ | testing | In-memory Mock (+ streaming) |
| `pkg/ai/genai/llm/chains` | ✅ | agent-orchestrator | LangChain-style chains |
| `pkg/ai/genai/llm/memory` | ✅ | context-manager | Conversation History (context-first) |
| `pkg/ai/nlp/rag` | ✅ | rag-service | Retrieval Augmented Generation |
| `pkg/ai/genai/llm/tools` | ✅ | agent-runtime | Function Calling/Tool Registry |
| `pkg/ai/nlp/embedding` | ✅ | embedding-service | Embedding Generation |

### Machine Learning (`pkg/ai/ml`)
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/ai/ml/training` | ✅ | training-job | Training Protocol Abstractions |
| `pkg/ai/ml/training/adapters/tensorflow`| ✅ | training-job | TensorFlow Training |
| `pkg/ai/ml/training/adapters/pytorch` | ✅ | training-job | PyTorch Training |
| `pkg/ai/ml/inference` | ✅ | inference-service | Model Serving Interface |
| `pkg/ai/ml/feature` | ✅ | feature-store | Feature Store Client |
| `pkg/ai/ml/sagemaker` | ✅ | training-job | AWS SageMaker Adapter |
| `pkg/ai/ml/vertexai` | ✅ | training-job | GCP Vertex AI Adapter |
| `pkg/ai/ml/azureml` | ✅ | training-job | Azure ML Adapter |
| `pkg/ai/ml/mlflow` | ✅ | model-registry | MLflow Adapter |

### Perception (`pkg/ai/perception`)
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/ai/perception/vision` | ✅ | media-analysis | Image Classification/OCR |
| `pkg/ai/perception/speech` | ✅ | transcription | STT / TTS |
| `pkg/ai/perception/ocr` | ✅ | document-parser | Document Intelligence |
| `pkg/ai/perception/vision/adapters/rekognition` | ✅ | media-analysis | AWS Rekognition Adapter |
| `pkg/ai/perception/speech/adapters/openai` | ✅ | transcription | OpenAI Whisper Adapter |

### NLP (`pkg/ai/nlp`)
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/ai/nlp/embedding` | ✅ | semantic-search | Text Embeddings Interface |
| `pkg/ai/nlp/embedding/adapters/openai` | ✅ | semantic-search | OpenAI Embeddings |
| `pkg/ai/nlp/embedding/adapters/huggingface` | ✅ | semantic-search | HF Inference Embeddings |
| `pkg/ai/nlp/embedding/adapters/memory` | ✅ | testing | In-memory Embeddings |
| `pkg/ai/nlp/rag` | ✅ | knowledge-bot | RAG Orchestrator |

### Generative AI (`pkg/ai/genai`)
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/ai/genai/llm` | ✅ | llm-core | LLM Client (`Chat` + `StreamChat`) + errors/instrumented |
| `pkg/ai/genai/llm/adapters/openai` | ✅ | llm-core | OpenAI Adapter |
| `pkg/ai/genai/llm/adapters/anthropic` | ✅ | llm-core | Anthropic Adapter |
| `pkg/ai/genai/llm/adapters/gemini` | ✅ | llm-core | Google Gemini Adapter |
| `pkg/ai/genai/llm/adapters/ollama` | ✅ | llm-core | Ollama Adapter (Local LLM) |
| `pkg/ai/genai/llm/adapters/memory` | ✅ | testing | In-memory Mock (+ streaming) |
| `pkg/ai/nlp/embedding` | ✅ | embedding-service | Embedding Generation (canonical; not under genai/llm) |
| `pkg/ai/nlp/rag` | ✅ | rag-service | Retrieval Augmented Generation (canonical) |
| `pkg/ai/genai/llm/memory` | ✅ | context-manager | Conversation History (context-first) |
| `pkg/ai/genai/llm/chains` | ✅ | agent-orchestrator | LangChain-style chains |
| `pkg/ai/genai/llm/tools` | ✅ | agent-runtime | Function Calling/Tool Registry |
| `pkg/ai/genai/image` | ✅ | creative-tools | Image Generation Interface |
| `pkg/ai/genai/image/adapters/openai` | ✅ | creative-tools | DALL-E Adapter |
| `pkg/ai/genai/image/adapters/memory` | ✅ | testing | In-memory Image Generation |
| `pkg/ai/genai/agents` | ✅ | autonomous-tasks| ReAct Agent Framework |

---

## 2. Communication (`pkg/communication`)

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/communication/email` | ✅ | notification | Email Interface |
| `pkg/communication/email/adapters/sendgrid`| ✅ | notification | SendGrid Adapter |
| `pkg/communication/email/adapters/ses` | ✅ | notification | AWS SES Adapter |
| `pkg/communication/email/adapters/smtp` | ✅ | notification | Standard SMTP Adapter |
| `pkg/communication/sms` | ✅ | notification | SMS Interface |
| `pkg/communication/sms/adapters/twilio` | ✅ | notification | Twilio Adapter |
| `pkg/communication/sms/adapters/sns` | ✅ | notification | AWS SNS Adapter |
| `pkg/communication/push` | ✅ | push-service | Push Notification Interface |
| `pkg/communication/push/adapters/fcm` | ✅ | push-service | Firebase Cloud Messaging |
| `pkg/communication/push/adapters/apns` | ✅ | push-service | Apple Push Notification |
| `pkg/communication/chat` | ✅ | chatbot | Chat Platform Integrations (Slack/Discord) |
| `pkg/communication/template` | ✅ | notification | Production Template Engine |

---

## 3. Commerce (`pkg/commerce`)

> 🔄 Improved — Money + payment webhooks/auth-capture/idempotency/events; billing plans; tax multi-jurisdiction memory. TaxJar/Avalara + live FX still open. See [`MISSING_CAPABILITIES.md`](../MISSING_CAPABILITIES.md).

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/commerce` | ✅ | — | Shared `Money` (int64 minor units) |
| `pkg/commerce/payment` | ✅ | payment-gateway | Provider + Authorizer + webhooks + Evented/Resilient |
| `pkg/commerce/payment/adapters/stripe` | ✅ | payment-gateway | Stripe + webhook verify + resilience |
| `pkg/commerce/payment/adapters/paypal` | ✅ | payment-gateway | PayPal + webhook verify + resilience |
| `pkg/commerce/billing` | 🔄 | billing-engine | Plans/upgrade/past_due; proration stub |
| `pkg/commerce/tax` | 🔄 | tax-service | Multi-jurisdiction memory; TaxJar planned |
| `pkg/commerce/currency` | 🔄 | currency-exchange | Static FX + FormatMoney; live feed interface only |

---

## 4. Data & Analytics (`pkg/data`, `pkg/bigdata`)

### Big Data (`pkg/bigdata`)
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/data/bigdata/formats/avro` | ✅ | data-ingestion | Avro Format Support |
| `pkg/data/bigdata/formats/parquet` | ✅ | data-ingestion | Parquet Format Support |
| `pkg/data/bigdata/compute/spark` | 🔄 | big-data-job | Local spark-submit wrapper (Spark Connect planned) |
| `pkg/data/bigdata/compute/mapreduce` | ✅ | big-data-job | MapReduce Implementation |
| `pkg/data/bigdata/olap/duckdb` | ✅ | analytics | Embedded OLAP (DuckDB) |
| `pkg/data/bigdata/adapters/bigquery` | ✅ | analytics | GCP BigQuery Adapter |
| `pkg/data/bigdata/adapters/redshift` | ✅ | analytics | AWS Redshift Adapter |
| `pkg/data/bigdata/adapters/synapse` | ✅ | analytics | Azure Synapse Adapter |
| `pkg/data/bigdata/lake/hdfs` | ✅ | storage | HDFS Client |
| `pkg/data/bigdata/pipeline/dag` | ✅ | workflow | DAG Executor |
| `pkg/data/bigdata/pipeline/etl` | ✅ | etl | ETL Pipeline Framework |

### Database (`pkg/database`)

#### SQL
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/database/sql/adapters/postgres` | ✅ | relational-db | PostgreSQL Adapter |
| `pkg/database/sql/adapters/mysql` | ✅ | relational-db | MySQL Adapter |
| `pkg/database/sql/adapters/sqlite` | ✅ | relational-db | SQLite Adapter |
| `pkg/database/sql/adapters/mssql` | ✅ | relational-db | SQL Server Adapter |
| `pkg/database/sql/adapters/clickhouse` | ✅ | analytics-db | ClickHouse Adapter |

#### NoSQL
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/database/timeseries` | ✅ | telemetry | Time-series Interface |
| `pkg/database/timeseries/adapters/timestream`| ✅ | telemetry | AWS Timestream Adapter |
| `pkg/database/timeseries/adapters/influxdb` | ✅ | telemetry | InfluxDB Adapter |
| `pkg/database/document` | ✅ | cms | Document DB Interface |
| `pkg/database/document/adapters/dynamodb` | ✅ | highly-scalable | AWS DynamoDB Adapter |
| `pkg/database/document/adapters/cosmosdb` | ✅ | multi-region | Azure CosmosDB Adapter |
| `pkg/database/document/adapters/firestore` | ✅ | mobile-backend | GCP Firestore Adapter |
| `pkg/database/document/adapters/mongodb` | ✅ | document-store | MongoDB Adapter |
| `pkg/database/graph` | ✅ | recommendation | Graph DB Interface |
| `pkg/database/kv/adapters/redis` | ✅ | cache/kv | Redis KV Adapter |

### Storage (File/Block/Object)
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/storage/blob` | ✅ | media-store | Object Storage (S3/GCS/Azure/local/memory) |
| `pkg/storage/file` | 🔄 | shared-fs | Interface + memory only (EFS/NFS not implemented) |
| `pkg/storage/block` | 🔄 | vm-disk | Interface + memory only (EBS not implemented) |
| `pkg/storage/archive` | 🔄 | backup | Cold storage interface + memory only (Glacier not implemented) |

### Search
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/data/search` | ✅ | resource-search | Search Interface |
| `pkg/data/search/adapters/elasticsearch` | ✅ | resource-search | Elasticsearch Adapter |
| `pkg/data/search/adapters/meilisearch` | ✅ | resource-search | Meilisearch Adapter |
| `pkg/data/search/adapters/algolia` | ✅ | resource-search | Algolia Adapter |

---

## 5. Workflows & Orchestration (`pkg/workflow`)

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/workflow` | ✅ | state-machine | Workflow Engine Interface + Memory Adapter |
| `pkg/workflow/adapters/stepfunctions` | ✅ | state-machine | AWS Step Functions |
| `pkg/workflow/adapters/temporal` | ✅ | durable-execution| Temporal Client |
| `pkg/workflow/adapters/logicapps` | ✅ | integration | Azure Logic Apps |
| `pkg/workflow/saga` | ✅ | order-manager | Saga Pattern Orchestrator |
| `pkg/workflow/scheduler` | ✅ | cron-service | Distributed Job Scheduler |

---

## 6. Security & Identity (`pkg/security`, `pkg/auth`)

### Auth
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/auth/adapters/cognito` | ✅ | identity-provider| AWS Cognito Adapter |
| `pkg/auth/adapters/gcpidentity` | ✅ | identity-provider| GCP Identity Adapter |
| `pkg/auth/adapters/entraid` | ✅ | identity-provider| Azure Entra ID Adapter |
| `pkg/auth/session` | ✅ | api-gateway | Distributed Session Management |
| `pkg/auth/mfa` | ✅ | auth-service | Multi-Factor Authentication |
| `pkg/auth/webauthn` | ✅ | auth-service | Passkeys / Biometrics |

### Protection
> 🔄 Foundation raised (root/errors, crypto, secrets Rotate, reCAPTCHA). Still no Vault/cloud KMS/WAF/scanner production backends. See [`MISSING_CAPABILITIES.md`](../MISSING_CAPABILITIES.md#pkgsecurity-30--improved).

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/security/fraud` | 🔄 | checkout | Fraud Detection/Risk Scoring (memory) |
| `pkg/security/captcha` | 🔄 | registration | Bot Protection (memory) |
| `pkg/security/waf` | 🔄 | edge-security | Web Application Firewall Control (memory) |
| `pkg/security/crypto/kms` | 🔄 | key-management | Key Management Service (memory) |
| `pkg/security/secrets` | 🔄 | vault | Secret Management Interface (memory) |
| `pkg/security/scanning` | 🔄 | compliance | Vulnerability Scanning (memory; GuardDuty not wired) |

---

## 7. Core Infrastructure (`pkg/network`, `pkg/compute`)

### Networking
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/network/loadbalancer` | ✅ | private-cloud | Load Balancer Interface + Memory Adapter |
| `pkg/network/loadbalancer/adapters/aws`| ✅ | cloud-infra | AWS ELB/ALB Management |
| `pkg/network/loadbalancer/adapters/gcp`| ✅ | cloud-infra | GCP Load Balancing |
| `pkg/network/dns` | ✅ | service-discovery| DNS Management Interface + Memory Adapter |
| `pkg/network/cdn` | ✅ | content-delivery | CDN Management Interface + Memory Adapter |
| `pkg/network/apigateway` | ✅ | api-routing | API Gateway Interface + Memory Adapter |
| `pkg/network/ip` | ✅ | geo-blocking | IP Intelligence Interface + Memory Adapter |

### Compute
> 🔄 VM has interface + memory only (no EC2/GCE/Azure adapters). See [`MISSING_CAPABILITIES.md`](../MISSING_CAPABILITIES.md#pkgcompute-52).

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/compute/vm` | 🔄 | iaas | VM Management Interface + Memory Adapter (no cloud adapters) |
| `pkg/compute/container` | ✅ | paas | Container Runtime Interface + Memory Adapter |
| `pkg/compute/serverless` | ✅ | faas | Serverless Runtime Interface + Memory Adapter |
| `pkg/compute/serverless/adapters/lambda` | ✅ | faas | AWS Lambda Management |
| `pkg/compute/serverless/adapters/gcf` | ✅ | faas | Google Cloud Functions |
| `pkg/compute/container/adapters/k8s` | ✅ | paas | Kubernetes Client/Controller |
| `pkg/compute/container/adapters/fargate` | ✅ | paas | AWS Fargate |

---

## 8. Web3 (`pkg/web3`)

> ✅ Root Client/Store/Verifier interfaces, memory adapters, instrumentation, race-safe SIWE; ethereum/solana/ipfs scaffolds remain SDK-coupled. See [`MISSING_CAPABILITIES.md`](../MISSING_CAPABILITIES.md#pkgweb3-22).

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/web3` | ✅ | — | Client/Store/Verifier interfaces, errors, instrumented wrappers |
| `pkg/web3/adapters/memory` | ✅ | — | In-memory Ethereum, IPFS, SIWE adapters |
| `pkg/web3/identity` | ✅ | auth-dapp | SIWE crypto verify (race-safe nonces); DID parse/format only |
| `pkg/web3/blockchain/ethereum` | 🔄 | wallet | geth ethclient wrapper (not yet behind root Client) |
| `pkg/web3/blockchain/solana` | 🔄 | wallet | Solana JSON-RPC scaffold (no root interface yet) |
| `pkg/web3/storage/ipfs` | 🔄 | nft-storage | IPFS HTTP API scaffold (not yet behind Store) |

---

## 9. IoT (`pkg/iot`)

> 🔄 Root interfaces + memory adapters + tests for MQTT/OTA; AWS adapters still SDK-coupled (not behind root Client). No CoAP/device registry. See [`MISSING_CAPABILITIES.md`](../MISSING_CAPABILITIES.md#pkgiot-28).

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/iot` | ✅ | — | Client/Updater interfaces, errors, instrumented, semver helpers |
| `pkg/iot/adapters/memory` | ✅ | — | In-memory MQTT + OTA adapters |
| `pkg/iot/protocols/mqtt` | 🔄 | vehicle-telemetry| Paho MQTT client (timeout handling fixed; not wrapped as root Client) |
| `pkg/iot/device/ota` | 🔄 | device-manager | HTTP OTA (semver + resilience retry; ApplyUpdate stub) |
| `pkg/iot/adapters/awsiot` | 🔄 | iot-cloud | AWS IoT Core SDK wrapper (not behind root Client) |
| `pkg/iot/adapters/greengrass` | 🔄 | edge-compute | AWS Greengrass V2 management SDK wrapper |

---

## 10. Enterprise Patterns (`pkg/enterprise`)

> 🔄 Design stubs; 0 tests; not standards-complete. See [`MISSING_CAPABILITIES.md`](../MISSING_CAPABILITIES.md#pkgenterprise-24).

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/enterprise/ddd` | 🔄 | core-business | Domain-Driven Design Primitives (stub) |
| `pkg/enterprise/cqrs` | 🔄 | reporting | Command Query Responsibility Segregation (stub) |
| `pkg/enterprise/eventsource` | 🔄 | audit-log | Event Sourcing Store (in-memory only) |

---

## 11. Private Cloud Components (Infrastructure-as-a-Service)

> **MISSING REQUIREMENTS**: To build a "Private Cloud" (AWS equivalent) on bare metal, you need the following **Server-Side** capabilities, not just clients.
>
> 🔄 Cloud packages are a **memory-only IaaS scaffold** today (no Libvirt/Firecracker/IPMI/PXE). Metering is memory-only with 0 tests. See [`MISSING_CAPABILITIES.md`](../MISSING_CAPABILITIES.md#pkgcloud-38) and [`metering`](../MISSING_CAPABILITIES.md#pkgmetering-20).

| Domain | Package | Needs Implementation | Description |
|--------|---------|---------------------|-------------|
| **Compute** | `pkg/cloud/hypervisor` | 🔄 | VM Management interface + memory (Libvirt/QEMU/Firecracker not wired) |
| **Compute** | `pkg/cloud/provisioning` | 🔄 | Bare Metal Provisioning interface + memory (PXE/IPMI not wired) |
| **Compute** | `pkg/cloud/scheduler` | 🔄 | Placement Logic interface + memory (bin-packing strategies stubby) |
| **Network** | `pkg/network/sdn` | 🔄 | Software Defined Networking (VPC/Overlay) — scaffold |
| **Network** | `pkg/network/dhcp` | 🔄 | IP Address Management System (IPAM) — scaffold |
| **Network** | `pkg/network/firewall` | 🔄 | Distributed Firewall / Security Groups — scaffold |
| **Storage** | `pkg/storage/controller` | 🔄 | Volume Controller (Ceph/LVM wrapper) — scaffold |
| **Identity** | `pkg/security/iam/provider` | 🔄 | Identity Provider Server (OIDC/SAML issuer) — scaffold |
| **Billing** | `pkg/metering` | 🔄 | Usage Metering & Rating Engine (memory only; 0 tests) |
| **Control** | `pkg/cloud/controlplane` | 🔄 | API Server & State Manager (memory scaffold) |

