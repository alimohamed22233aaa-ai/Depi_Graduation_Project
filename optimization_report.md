# Online Boutique Containerization and Image Optimization Report

## Executive summary

The application is a polyglot microservices demo consisting of Go, .NET, Java,
Node.js, and Python services, plus Redis and PostgreSQL. It originally had no
complete local Docker deployment.

The repository now has a Docker Compose deployment with one image per
application service. Only the frontend is exposed publicly on port `8080`;
the internal services communicate over the Compose network.

The main optimization goal was reducing local disk usage while preserving the
existing service contracts and runtime behavior.

## Image size progression

The following figures are approximate Docker image sizes measured during the
work.

| Service | Initial image | Final image | Reduction |
|---|---:|---:|---:|
| `adservice` | 486 MB | 300 MB | 186 MB |
| `cartservice` | 351 MB | 197 MB | 154 MB |
| `ordersservice` | 243 MB | 142 MB | 101 MB |
| `emailservice` | 232 MB | 141 MB | 91 MB |
| `recommendationservice` | 229 MB | 138 MB | 91 MB |
| `searchservice` | 188 MB | 93.9 MB | 94 MB |
| `currencyservice` | 509 MB | 387 MB | 122 MB |
| `paymentservice` | 498 MB | 376 MB | 122 MB |
| `frontend` | — | 43.7 MB | — |
| `productcatalogservice` | — | 41.5 MB | — |
| `checkoutservice` | — | 31.4 MB | — |
| `shippingservice` | — | 30.6 MB | — |

The directly measured reductions total approximately **961 MB** across the
optimized application images. Docker layer sharing means the actual disk
saving is lower than the arithmetic sum, but the active image footprint is
now substantially smaller.

## Optimizations applied

### Go services

Go services use multi-stage builds with:

- `CGO_ENABLED=0`
- stripped symbols using `-s -w`
- `-trimpath`
- Distroless static runtime images
- non-root runtime users

This produced small images between approximately 30 and 44 MB.

### Java ad service

The build still uses a full JDK, but the runtime now uses:

```dockerfile
gcr.io/distroless/java21-debian12:nonroot
```

Only the compiled dependency libraries are copied into the final image. This
removed the full Java runtime distribution and reduced the image from 486 MB
to 300 MB.

### .NET cart service

The build stage remains on the standard .NET SDK image. The runtime was changed
to:

```dockerfile
mcr.microsoft.com/dotnet/aspnet:10.0-alpine
```

The service builds, starts, connects to Redis, and serves gRPC correctly. The
image dropped from 351 MB to 197 MB.

### Node.js services

Payment and currency services use Distroless Node.js runtime images. The final
images contain only production dependencies and application files. This saved
approximately 122 MB per service.

The currency service also required preserving the `data/` directory structure;
flattening it caused the runtime error for
`./data/currency_conversion.json`.

### Python services

All four Python services now use `python:3.12-alpine` with Alpine-compatible
wheels. This reduced the four images by approximately 377 MB combined.

The runtime copies were also narrowed so that `emailservice` and
`recommendationservice` ship only the files required by their servers rather
than the complete build context.

### Build and Compose improvements

- Per-service `.dockerignore` files prevent virtual environments, caches,
  bytecode, build output, and test artifacts from entering build contexts.
- Dependency files are copied before source files to improve Docker layer
  reuse.
- Redis and PostgreSQL have healthchecks.
- Compose dependency restart propagation was added to reduce stale gRPC
  connections after a dependency container is recreated.
- Only frontend port `8080` is public by default.