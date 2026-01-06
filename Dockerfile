FROM python:3.12-alpine

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

WORKDIR /app

RUN apk add --no-cache tzdata \
    && pip install --no-cache-dir croniter==2.0.5

COPY src/main.py /app/main.py

USER nobody

CMD ["python", "/app/main.py"]
