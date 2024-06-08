# docker buildx build --platform linux/arm64,linux/amd64 --tag ghcr.io/juho05/crossonic-sonos-controller:latest --push .
FROM python:3-alpine

WORKDIR /app
COPY main.py requirements.txt /app

RUN pip3 install --upgrade pip && pip install --no-cache-dir -r requirements.txt

ENV HOST "127.0.0.1"
ENV PORT 8257

EXPOSE 8257

CMD ["sh", "-c", "echo Listening at: http://${HOST}:${PORT} && gunicorn main:app --log-level warning -b ${HOST}:${PORT} -w 4"]
