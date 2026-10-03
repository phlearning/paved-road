"""HTTP metrics with the names and labels the platform dashboard expects."""

import time

from fastapi import FastAPI, Request, Response
from prometheus_client import CONTENT_TYPE_LATEST, Counter, Histogram, generate_latest

REQUESTS = Counter("http_requests_total", "HTTP requests handled.", ["method", "handler", "status"])
LATENCY = Histogram("http_request_duration_seconds", "HTTP request latency.", ["method", "handler"])

# Probes and scrapes would drown the business traffic.
UNTRACKED = {"/metrics", "/healthz", "/readyz"}


def instrument(app: FastAPI) -> None:
    @app.middleware("http")
    async def record(request: Request, call_next):
        start = time.perf_counter()
        status = 500
        try:
            response = await call_next(request)
            status = response.status_code
            return response
        finally:
            # The route template, not the raw path, keeps label cardinality bounded.
            route = request.scope.get("route")
            handler = getattr(route, "path", "unmatched")
            if handler not in UNTRACKED:
                REQUESTS.labels(request.method, handler, f"{status // 100}xx").inc()
                LATENCY.labels(request.method, handler).observe(time.perf_counter() - start)

    @app.get("/metrics", include_in_schema=False)
    def metrics() -> Response:
        return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)
