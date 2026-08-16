# ZiadaPOS MCP agent — deployed as its own service, talking to the API over
# HTTP and authenticating with the shop-owner JWT (login + refresh).
FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim AS builder
WORKDIR /app
COPY pyproject.toml uv.lock README.md ./
COPY src ./src
RUN uv sync --frozen --no-dev --no-install-project \
    && uv build --wheel

FROM python:3.12-slim
WORKDIR /app
COPY --from=builder /app/dist/*.whl /tmp/
RUN pip install --no-cache-dir /tmp/*.whl \
    && rm -rf /tmp/*.whl

RUN addgroup --system --gid 1001 ziada \
    && adduser --system --uid 1001 --ingroup ziada ziada
USER ziada

EXPOSE 8081
ENV MCP_TRANSPORT=http \
    MCP_HOST=0.0.0.0 \
    MCP_PORT=8081
CMD ["ziada-mcp"]
