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
> 🔄 Training/inference/feature are memory or local-subprocess depth; cloud trainers exist but are not production-hardened.

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/ai/ml/training` | ✅ | training-job | Training Protocol Abstractions |
| `pkg/ai/ml/training/adapters/tensorflow`| 🔄 | training-job | Local subprocess TensorFlow trainer (not TF Serving) |
| `pkg/ai/ml/training/adapters/pytorch` | 🔄 | training-job | Local subprocess PyTorch trainer |
| `pkg/ai/ml/inference` | 🔄 | inference-service | Interface + memory simulate (no real model runtime) |
| `pkg/ai/ml/feature` | 🔄 | feature-store | Interface + memory store (Feast/Redis backends reserved) |
| `pkg/ai/ml/sagemaker` | 🔄 | training-job | AWS SageMaker StartJob/Describe (depth varies) |
| `pkg/ai/ml/vertexai` | 🔄 | training-job | GCP Vertex AI adapter (depth varies) |
| `pkg/ai/ml/azureml` | 🔄 | training-job | Azure ML adapter (depth varies) |
| `pkg/ai/ml/mlflow` | 🔄 | model-registry | MLflow tracking HTTP client (partial registry surface) |

### Perception (`pkg/ai/perception`)
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/ai/perception/vision` | 🔄 | media-analysis | Interface + memory; cloud depth varies |
| `pkg/ai/perception/speech` | 🔄 | transcription | STT/TTS + OpenAI/AWS/Google (polish varies) |
| `pkg/ai/perception/ocr` | 🔄 | document-parser | Interface + memory + Textract |
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

> 🔄 Improved — Money + payment webhooks/auth-capture/idempotency/events; billing plans/proration/dunning; TaxJar/Avalara + live FX adapters.

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/commerce` | ✅ | — | Shared `Money` (int64 minor units) |
| `pkg/commerce/payment` | ✅ | payment-gateway | Provider + Authorizer + webhooks + Evented/Resilient |
| `pkg/commerce/payment/adapters/stripe` | ✅ | payment-gateway | Stripe + webhook verify + resilience |
| `pkg/commerce/payment/adapters/paypal` | ✅ | payment-gateway | PayPal + webhook verify + resilience |
| `pkg/commerce/billing` | 🔄 | billing-engine | Plans/upgrade/past_due/proration/dunning (depth varies) |
| `pkg/commerce/tax` | 🔄 | tax-service | Multi-jurisdiction memory + TaxJar/Avalara HTTP |
| `pkg/commerce/currency` | 🔄 | currency-exchange | Static FX + live feed adapters (OpenExchangeRates) |

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
| `pkg/storage/file` | 🔄 | shared-fs | Interface + memory + local FS (EFS/NFS cloud not wired) |
| `pkg/storage/block` | 🔄 | vm-disk | Interface + memory + local + EBS (SDK waiters; not full EC2 surface) |
| `pkg/storage/archive` | 🔄 | backup | Cold storage: memory + filesystem + Glacier + Azure/GCS archive |

### Search
| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/data/search` | ✅ | resource-search | Search Interface |
| `pkg/data/search/adapters/elasticsearch` | ✅ | resource-search | Elasticsearch Adapter |
| `pkg/data/search/adapters/opensearch` | ✅ | resource-search | OpenSearch HTTP client |
| `pkg/data/search/adapters/meilisearch` | ✅ | resource-search | Meilisearch Adapter |
| `pkg/data/search/adapters/algolia` | ✅ | resource-search | Algolia Adapter |
| `pkg/data/search/adapters/typesense` | ✅ | resource-search | Typesense HTTP client |

---

## 5. Workflows & Orchestration (`pkg/workflow`)

> 🔄 Memory engine + durable saga/scheduler are solid; cloud adapters are thin SDK wrappers (depth varies).

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/workflow` | ✅ | state-machine | Workflow Engine Interface + Memory Adapter |
| `pkg/workflow/adapters/stepfunctions` | 🔄 | state-machine | AWS Step Functions (thin; completeness varies) |
| `pkg/workflow/adapters/temporal` | 🔄 | durable-execution| Temporal Client (thin; completeness varies) |
| `pkg/workflow/adapters/logicapps` | ✅ | integration | Azure Logic Apps (ARM + MSI/client-secret/DefaultAzureCredential) |
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
> 🔄 Vault KV v2, AWS/GCP/Azure KMS + secret managers, Cloudflare+AWS WAF, GuardDuty + ClamAV landed; fraud/captcha remain memory-depth.

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/security/fraud` | 🔄 | checkout | Fraud Detection/Risk Scoring (memory) |
| `pkg/security/captcha` | 🔄 | registration | Bot Protection (memory + reCAPTCHA) |
| `pkg/security/waf` | 🔄 | edge-security | WAF (memory + Cloudflare + AWS WAFv2 IPSet) |
| `pkg/security/crypto/kms` | 🔄 | key-management | KMS (memory + AWS/GCP/Azure Encrypt/Decrypt) |
| `pkg/security/secrets` | 🔄 | vault | Secrets (memory + Vault KV v2 + AWS/GCP/Azure SM) |
| `pkg/security/scanning` | 🔄 | compliance | Scanning (memory + GuardDuty findings + ClamAV INSTREAM) |

---

## 7. Core Infrastructure (`pkg/network`, `pkg/compute`)

### Networking
> 🔄 CDN/APIGW/IP/DNS are interface + memory only (Route53/CloudFront/etc. reserved). LB has AWS/GCP adapters.

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/network/loadbalancer` | ✅ | private-cloud | Load Balancer Interface + Memory Adapter |
| `pkg/network/loadbalancer/adapters/aws`| ✅ | cloud-infra | AWS ELB/ALB Management |
| `pkg/network/loadbalancer/adapters/gcp`| ✅ | cloud-infra | GCP Load Balancing |
| `pkg/network/dns` | 🔄 | service-discovery| DNS Interface + Memory (Route53/CloudDNS reserved) |
| `pkg/network/cdn` | 🔄 | content-delivery | CDN Interface + Memory (CloudFront/etc. reserved) |
| `pkg/network/apigateway` | 🔄 | api-routing | API Gateway Interface + Memory (AWS/Kong reserved) |
| `pkg/network/ip` | 🔄 | geo-blocking | IP Intelligence + Memory (MaxMind/etc. reserved) |

### Compute
> 🔄 VM: memory + EC2/GCE; Azure VM is Unimplemented scaffold. Serverless: memory + Lambda/GCF; Azure Functions HTTP Invoke + ARM CRUD Unimplemented.

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/compute/vm` | 🔄 | iaas | VM interface + memory + EC2/GCE; Azure VM Unimplemented scaffold |
| `pkg/compute/container` | ✅ | paas | Container Runtime + memory + resilient wrapper |
| `pkg/compute/serverless` | ✅ | faas | Serverless Runtime Interface + Memory Adapter |
| `pkg/compute/serverless/adapters/lambda` | ✅ | faas | AWS Lambda Management |
| `pkg/compute/serverless/adapters/gcf` | ✅ | faas | Google Cloud Functions |
| `pkg/compute/serverless/adapters/azurefunctions` | 🔄 | faas | Azure Functions (HTTP Invoke; ARM CRUD Unimplemented) |
| `pkg/compute/container/adapters/k8s` | 🔄 | paas | Kubernetes (Create ID = pod name; Exec SPDY; Stats Unimplemented without metrics-server) |
| `pkg/compute/container/adapters/fargate` | ✅ | paas | AWS Fargate |

---

## 8. Web3 (`pkg/web3`)

> ✅ Root Client/Store/Verifier/SolanaClient + memory; geth/kubo/solana adapters; ethereum/ipfs/solana thin re-exports; WalletConnect is session stub (no relay).

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/web3` | ✅ | — | Client/Store/Verifier/SolanaClient interfaces, errors, instrumented wrappers |
| `pkg/web3/adapters/memory` | ✅ | — | In-memory Ethereum, IPFS, SIWE, Solana, WalletConnect stubs |
| `pkg/web3/adapters/geth` | ✅ | wallet | go-ethereum ethclient behind web3.Client |
| `pkg/web3/adapters/kubo` | ✅ | nft-storage | Kubo HTTP API behind web3.Store |
| `pkg/web3/adapters/solana` | ✅ | wallet | Solana JSON-RPC behind web3.SolanaClient |
| `pkg/web3/identity` | ✅ | auth-dapp | SIWE verify; DID parse/format + memory ethr/web resolvers |
| `pkg/web3/blockchain/ethereum` | ✅ | wallet | Thin wrapper → adapters/geth |
| `pkg/web3/blockchain/solana` | ✅ | wallet | Thin wrapper → adapters/solana |
| `pkg/web3/storage/ipfs` | ✅ | nft-storage | Thin wrapper → adapters/kubo |

---

## 9. IoT (`pkg/iot`)

> 🔄 Root interfaces + memory; awsiot/greengrass/mqtt behind root Client; CoAP UDP + device registry + cert helpers.

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/iot` | ✅ | — | Client/Updater interfaces, errors, instrumented, semver helpers |
| `pkg/iot/adapters/memory` | ✅ | — | In-memory MQTT + OTA adapters |
| `pkg/iot/adapters/mqtt` | ✅ | vehicle-telemetry| Paho MQTT behind root `iot.Client` |
| `pkg/iot/protocols/mqtt` | 🔄 | vehicle-telemetry| Paho MQTT protocol client (prefer adapters/mqtt for root Client) |
| `pkg/iot/protocols/coap` | 🔄 | edge | CoAP Memory stub + UDP datagram (Observe/DTLS still limited) |
| `pkg/iot/device/ota` | 🔄 | device-manager | HTTP OTA + blob-backed updater (ApplyUpdate platform stub) |
| `pkg/iot/device/registry` | ✅ | device-manager | DeviceRegistry interface + memory |
| `pkg/iot/device/cert` | ✅ | device-manager | Device cert types + memory + awsiot CertificateProvider |
| `pkg/iot/adapters/awsiot` | ✅ | iot-cloud | AWS IoT + NewAdapter behind root Client |
| `pkg/iot/adapters/greengrass` | ✅ | edge-compute | Greengrass V2 management + NewAdapter behind root Client |

---

## 10. Enterprise Patterns (`pkg/enterprise`)

> 🔄 Design stubs + ProjectionRunner/checkpoint/outbox landed; not standards-complete.

| Package | Status | Enables Services | Description |
|---------|--------|------------------|-------------|
| `pkg/enterprise/ddd` | 🔄 | core-business | Domain-Driven Design Primitives (stub) |
| `pkg/enterprise/cqrs` | 🔄 | reporting | CQRS + ProjectionRunner (depth varies) |
| `pkg/enterprise/eventsource` | ✅ | audit-log | Event Sourcing (memory + SQL snapshots/checkpoints + messaging outbox) |

---

## 11. Private Cloud Components (Infrastructure-as-a-Service)

> **MISSING REQUIREMENTS**: To build a "Private Cloud" (AWS equivalent) on bare metal, you need the following **Server-Side** capabilities, not just clients.
>
> 🔄 Cloud packages have memory scaffolds plus Libvirt/Firecracker/Redfish/IPMI/PXE adapters (depth varies). Metering has Prometheus exporter.

| Domain | Package | Needs Implementation | Description |
|--------|---------|---------------------|-------------|
| **Compute** | `pkg/cloud/hypervisor` | 🔄 | VM Management + memory + remote libvirt / Firecracker |
| **Compute** | `pkg/cloud/provisioning` | 🔄 | Bare Metal + memory + PXE HTTP / Redfish/IPMI |
| **Compute** | `pkg/cloud/scheduler` | ✅ | Placement: binpack / spread / random (memory) |
| **Network** | `pkg/network/sdn` | 🔄 | SDN (VPC/Overlay) — interface + memory only (vxlan/calico reserved) |
| **Network** | `pkg/network/dhcp` | 🔄 | IPAM — interface + memory only (phpipam/netbox reserved) |
| **Network** | `pkg/network/firewall` | 🔄 | Distributed firewall / SG — interface + memory only (iptables/nftables reserved) |
| **Storage** | `pkg/storage/controller` | ✅ | Volume Controller (memory + LVM + Ceph RBD-shaped + CSI-shaped) |
| **Identity** | `pkg/security/iam/provider` | 🔄 | IdP server scaffold (memory only; no Dex/Keycloak; prefer pkg/auth) |
| **Billing** | `pkg/metering` | 🔄 | Usage Metering & Rating + Prometheus exporter |
| **Control** | `pkg/cloud/controlplane` | 🔄 | API Server & State Manager (memory + etcd HTTP) |

