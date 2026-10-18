# E-Commerce Microservices — application code only

Based on **Online Boutique** (https://github.com/GoogleCloudPlatform/microservices-demo,
Apache 2.0, Google LLC) plus two extra services written for this project.

This package contains **application code only**. All deployment/DevOps material was
removed on purpose: no Dockerfiles, no Kubernetes/Helm/Kustomize/Skaffold, no Terraform,
no CI/CD, no compose files. Those are the DevOps deliverables of the project.

See **SERVICES.md** for every service's language, port, environment variables and
build/run commands.

## Services (12 + Redis)

| Service | Language | Role |
|---|---|---|
| frontend | Go | web UI (HTTP) |
| productcatalogservice | Go | product list (reads products.json) |
| cartservice | C# (.NET) | shopping cart (stored in Redis) |
| checkoutservice | Go | places the order (calls payment, shipping, email, cart) |
| paymentservice | Node.js | mock payment |
| shippingservice | Go | shipping quote + tracking id |
| emailservice | Python | order confirmation (mock) |
| currencyservice | Node.js | currency conversion |
| recommendationservice | Python | recommended products |
| adservice | Java | contextual ads |
| **searchservice** (new) | Python/Flask | product search, REST |
| **ordersservice** (new) | Python/Flask | stores finished orders in SQL (PostgreSQL / RDS), REST |
| loadgenerator | Python/Locust | fake traffic (use it to demo autoscaling) |
| Redis | - | required by cartservice |

Internal calls between the original services use **gRPC** (definitions in `protos/`).

## What was changed vs. upstream
1. Removed: all DevOps files (listed above) and `shoppingassistantservice` (needs Google Cloud AI/AlloyDB; optional upstream feature).
2. `checkoutservice/main.go`: after a successful order it POSTs the order to `ordersservice`
   (`recordOrder`). Best-effort with a 2 s timeout, and it does nothing if `ORDERS_SERVICE_ADDR`
   is not set, so checkout never fails because of it.
3. Added `src/searchservice` and `src/ordersservice` (with unit tests).

## Known limits
- `searchservice` has its own copy of `products.json` (same file as productcatalogservice).
- Search is available at `GET /search` (small page) and `GET /api/search?q=...`; it is not integrated into the Go frontend UI.
- The modified `checkoutservice` Go code could not be compiled where this package was prepared; your first build will verify it.
