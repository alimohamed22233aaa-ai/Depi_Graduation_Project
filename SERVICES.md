# Services reference

"Listens on" is the port the process uses inside its container/host.
All original services use **gRPC** except `frontend` (HTTP).
Env vars marked **required** make the process crash/refuse to work when missing.

| Service | Language | Listens on | Environment variables | Health check |
|---|---|---|---|---|
| frontend | Go | `PORT` = 8080 | **required:** `PRODUCT_CATALOG_SERVICE_ADDR`, `CURRENCY_SERVICE_ADDR`, `CART_SERVICE_ADDR`, `RECOMMENDATION_SERVICE_ADDR`, `SHIPPING_SERVICE_ADDR`, `CHECKOUT_SERVICE_ADDR`, `AD_SERVICE_ADDR`, `SHOPPING_ASSISTANT_SERVICE_ADDR` (any dummy value, e.g. `unused:80`; the feature stays off unless `ENABLE_ASSISTANT=true`) | HTTP `GET /_healthz` |
| productcatalogservice | Go | `PORT` = 3550 | `PORT`; needs `products.json` in its working dir | gRPC health |
| cartservice | C# (.NET) | 7070 | `REDIS_ADDR` (e.g. `redis-cart:6379`) | gRPC health |
| checkoutservice | Go | `PORT` = 5050 | **required:** `PRODUCT_CATALOG_SERVICE_ADDR`, `SHIPPING_SERVICE_ADDR`, `PAYMENT_SERVICE_ADDR`, `EMAIL_SERVICE_ADDR`, `CURRENCY_SERVICE_ADDR`, `CART_SERVICE_ADDR`; optional: `ORDERS_SERVICE_ADDR` (e.g. `ordersservice:8080`) | gRPC health |
| paymentservice | Node.js | `PORT` = 50051 | `PORT`, `DISABLE_PROFILER=1` | gRPC health |
| shippingservice | Go | `PORT` (default 50051) | `PORT`, `DISABLE_PROFILER=1` | gRPC health |
| emailservice | Python | `PORT` (default 8080) | `PORT`, `DISABLE_PROFILER=1` | gRPC health |
| currencyservice | Node.js | `PORT` = 7000 | `PORT`, `DISABLE_PROFILER=1` | gRPC health |
| recommendationservice | Python | `PORT` = 8080 | `PRODUCT_CATALOG_SERVICE_ADDR`, `PORT`, `DISABLE_PROFILER=1` | gRPC health |
| adservice | Java | `PORT` = 9555 | `PORT` | gRPC health |
| searchservice | Python | choose (we use 8080) | optional `PRODUCTS_FILE` | HTTP `GET /health`, metrics `/metrics` |
| ordersservice | Python | choose (we use 8080) | **required:** `DATABASE_URL`, e.g. `postgresql+psycopg2://user:pass@host:5432/orders` | HTTP `GET /health`, metrics `/metrics` |
| loadgenerator | Python (Locust) | - | `FRONTEND_ADDR` (host:port), `USERS`, `RATE` | - |
| Redis | - | 6379 | - | - |

Addresses like `productcatalogservice:3550` are `host:port` of the *other* service
(its DNS name / Kubernetes Service name plus the port it is exposed on).
Note: if you expose a service on a different port than it listens on (e.g. emailservice
8080 exposed as 5000), the `*_ADDR` of its callers must use the **exposed** port.

## Who calls whom
```
frontend -> productcatalog, currency, cart, recommendation, shipping, checkout, ad
checkout -> productcatalog, shipping, payment, email, currency, cart, (orders)
recommendation -> productcatalog
cart -> Redis
orders -> PostgreSQL
```

## Build / run hints (to write your own Dockerfiles)
Run these from each service folder.

| Service | Build | Run |
|---|---|---|
| Go services (frontend, productcatalog, checkout, shipping) | `go mod download && CGO_ENABLED=0 go build -o server .` | `./server` (frontend also needs `templates/` and `static/` next to the binary) |
| paymentservice, currencyservice | `npm install --only=production` | `node index.js` (payment) / `node server.js` (currency) |
| emailservice | `pip install -r requirements.txt` | `python email_server.py` |
| recommendationservice | `pip install -r requirements.txt` | `python recommendation_server.py` |
| adservice | `./gradlew installDist` | `build/install/hipstershop/bin/AdService` |
| cartservice | `cd src && dotnet publish cartservice.csproj -c Release -o /out` | `/out/cartservice` (or `dotnet run`) |
| searchservice | `pip install -r requirements.txt` | `gunicorn -b 0.0.0.0:8080 app:app` |
| ordersservice | `pip install -r requirements.txt` | `gunicorn -b 0.0.0.0:8080 "app:app_factory()"` |
| loadgenerator | `pip install -r requirements.txt` | `locust --host=http://$FRONTEND_ADDR --headless -u 10 -r 1` |

Upstream toolchain versions used by the original project (check upstream Dockerfiles for the exact tags):
Go 1.2x, Node 2x, Python 3.1x, Java (Temurin JDK 2x), .NET 10.

## New services: API
**searchservice**
- `GET /api/search?q=<words>&category=<cat>` -> `{query, count, results:[{id,name,description,picture,categories,price_usd}]}`
- `GET /search` -> small HTML search page

**ordersservice**
- `POST /api/orders` `{order_id, user_id, email, tracking_id, currency, total, items:[{product_id, quantity}]}` -> 201 / 400 / 409
- `GET /api/orders[?user_id=]`, `GET /api/orders/<order_id>`

## Tests (already included for the two new services)
```bash
cd src/searchservice && pip install -r requirements.txt -r requirements-dev.txt && pytest
cd src/ordersservice && pip install -r requirements.txt -r requirements-dev.txt && pytest
```
The original cartservice, shippingservice and productcatalogservice have their own tests
(`dotnet test`, `go test ./...`).
