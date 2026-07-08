FROM docker.m.daocloud.io/library/python:3.11-slim-bookworm

ENV DEBIAN_FRONTEND=noninteractive \
    PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple \
    PIP_TRUSTED_HOST=pypi.tuna.tsinghua.edu.cn \
    NPM_CONFIG_REGISTRY=https://registry.npmmirror.com

RUN set -eux; \
    if [ -f /etc/apt/sources.list ]; then \
      sed -i \
        -e 's#http://deb.debian.org/debian#https://mirrors.tuna.tsinghua.edu.cn/debian#g' \
        -e 's#http://security.debian.org/debian-security#https://mirrors.tuna.tsinghua.edu.cn/debian-security#g' \
        /etc/apt/sources.list; \
    fi; \
    if [ -f /etc/apt/sources.list.d/debian.sources ]; then \
      sed -i \
        -e 's#http://deb.debian.org/debian#https://mirrors.tuna.tsinghua.edu.cn/debian#g' \
        -e 's#http://security.debian.org/debian-security#https://mirrors.tuna.tsinghua.edu.cn/debian-security#g' \
        /etc/apt/sources.list.d/debian.sources; \
    fi; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
      bash \
      ca-certificates \
      coreutils \
      curl \
      file \
      findutils \
      git \
      gzip \
      jq \
      less \
      nodejs \
      npm \
      procps \
      tar \
      unzip \
      wget; \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    python -m pip install --no-cache-dir --upgrade pip setuptools wheel; \
    python -m pip install --no-cache-dir \
      h11==0.16.0 \
      httpx==0.28.1 \
      numpy==2.3.1 \
      pdfplumber==0.11.7 \
      pillow==11.2.1 \
      python-docx==1.2.0

RUN mkdir -p /workspace /uploads /outputs /skills

WORKDIR /workspace

CMD ["sleep", "infinity"]
