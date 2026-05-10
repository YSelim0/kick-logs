FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV UV_LINK_MODE=copy

WORKDIR /app

RUN pip install --no-cache-dir uv

COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --group dev --no-install-project

COPY src ./src
COPY tests ./tests
COPY alembic.ini ./
COPY alembic ./alembic
RUN uv sync --frozen --group dev

EXPOSE 8000

CMD ["sh", "-c", "uv run alembic upgrade head && uv run uvicorn kick_logs.main:app --host 0.0.0.0 --port 8000"]
