FROM python:3.14-slim
WORKDIR /app
COPY apps/langchain/requirements.lock ./requirements.lock
RUN pip install --no-cache-dir -r requirements.lock && useradd -u 10001 -m wisp
COPY apps/langchain/server.py ./server.py
RUN chmod 644 /app/server.py
USER wisp
ENV LANGSMITH_TRACING=false LANGCHAIN_TRACING_V2=false
CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8091", "--no-access-log"]
