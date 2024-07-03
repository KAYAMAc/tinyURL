FROM golang

COPY . /app

WORKDIR /app

CMD ["go", "run", "server.go"]