# Stage 1: Build frontend
FROM --platform=$BUILDPLATFORM node:22-alpine AS fe-builder

WORKDIR /app

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache bash git dos2unix
RUN npm config set registry https://registry.npmmirror.com

COPY rush.json ./
RUN npm install -g @microsoft/rush

COPY frontend/ ./frontend/
COPY common/ ./common/
COPY scripts/ ./scripts/

RUN find . -name "*.sh" -type f -exec dos2unix {} \;
RUN chmod +x scripts/hooks/post-rush-install.sh && rm -rf /app/common/temp && rush install
RUN rush build --to @coze-studio/app

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git gcc libc-dev

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN go build -ldflags="-s -w" -o /app/openynet main.go

# Stage 3: Final image
FROM alpine:3.22.0

WORKDIR /app

RUN apk add --no-cache pax-utils python3 python3-dev bind-tools file deno curl

RUN apk add --no-cache --virtual .python-build-deps build-base py3-pip git && \
    python3 -m venv --copies --upgrade-deps /app/.venv && \
    . /app/.venv/bin/activate && \
    pip install urllib3==1.26.16 && \
    pip install --no-cache-dir h11==0.16.0 httpx==0.28.1 pillow==11.2.1 pdfplumber==0.11.7 python-docx==1.2.0 numpy==2.3.1 && \
    apk del .python-build-deps

# Copy Go binary
COPY --from=builder /app/openynet /app/openynet

# Copy Python scripts
COPY backend/infra/impl/document/parser/builtin/parse_pdf.py /app/parse_pdf.py
COPY backend/infra/impl/document/parser/builtin/parse_docx.py /app/parse_docx.py
COPY backend/infra/impl/coderunner/script/sandbox.py /app/sandbox.py

# Copy conf and default .env
COPY backend/conf /app/resources/conf/
COPY docker/.env.example /app/.env

# Copy frontend dist into static directory
COPY --from=fe-builder /app/frontend/apps/coze-studio/dist /app/resources/static/
# Copy SDK static assets (WebSDK JS etc.)
COPY backend/static/static/sdk /app/resources/static/static/sdk/

ENV PATH="/app/.venv/bin:${PATH}"

RUN chmod +x /app/parse_pdf.py /app/parse_docx.py && \
    find /app/.venv/bin -type f -exec chmod +x {} \;

EXPOSE 8888

CMD ["/app/openynet"]
