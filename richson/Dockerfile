FROM python:3.12-slim

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /usr/local/bin/

WORKDIR /app

# Copy dependency files
COPY pyproject.toml uv.lock ./

# Install dependencies using locked versions (no editable install in prod)
RUN uv pip install --system --no-cache .

# Copy application source
COPY src/ src/
COPY alembic/ alembic/
COPY alembic.ini .

EXPOSE 8001

CMD ["uvicorn", "richson.main:app", "--host", "0.0.0.0", "--port", "8001", "--workers", "1"]
