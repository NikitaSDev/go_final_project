FROM ubuntu:latest

WORKDIR /app

COPY final-project ./
COPY web ./web

EXPOSE 7540

ENV TODO_PORT=7540
ENV TODO_DBFILE=/app/scheduler.db

CMD ["./final-project"]