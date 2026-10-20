FROM python:3.12-alpine

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt \
    && adduser -D -u 10001 appuser
COPY email_server.py email_client.py logger.py demo_pb2.py demo_pb2_grpc.py ./
COPY templates ./templates
USER appuser
EXPOSE 8080
CMD ["python", "email_server.py"]
