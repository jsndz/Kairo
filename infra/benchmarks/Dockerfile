FROM golang:alpine

# Install Node.js, npm, bash, git, and util-linux (taskset)
RUN apk add --no-cache nodejs npm bash util-linux git

WORKDIR /workspace

ENV GOMAXPROCS=2
ENV GOTOOLCHAIN=auto

CMD ["./bench.sh", "report"]
