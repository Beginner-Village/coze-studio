# Intent Hub Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Intent Hub backend — a standalone Python service providing multi-agent intent routing with hierarchical skill disclosure, context management, and ChatFlow orchestration.

**Architecture:** FastAPI handles HTTP/SSE/management APIs. LangGraph handles intent classification graph with stateful checkpointing. MySQL stores persistent data, Redis caches session state. The project lives at `~/projects/ynet/intent-hub/` as a fully independent repo.

**Tech Stack:** Python 3.11+, FastAPI, LangGraph, LangChain, SQLAlchemy 2.x, Alembic, Redis, MySQL, sse-starlette, pytest

**Spec:** `docs/superpowers/specs/2026-04-16-intent-hub-design.md`

**Scope:** This plan covers the backend only. Frontend management UI is a separate plan.

---

## File Structure

### New Files (all paths relative to `~/projects/ynet/intent-hub/`)

| File | Responsibility |
|------|---------------|
| `pyproject.toml` | Project metadata, dependencies |
| `backend/main.py` | FastAPI app entry point |
| `backend/core/config.py` | Pydantic Settings for env-based config |
| `backend/core/database.py` | SQLAlchemy async engine + session factory |
| `backend/core/redis.py` | Redis connection pool |
| `backend/core/security.py` | Credential encryption/decryption (Fernet) |
| `backend/models/base.py` | SQLAlchemy declarative base, common mixins |
| `backend/models/chatflow_agent.py` | ChatflowAgent ORM model |
| `backend/models/skill_category.py` | SkillCategory ORM model |
| `backend/models/skill_mapping.py` | SkillChatflowMapping ORM model |
| `backend/models/model_config.py` | ModelConfig ORM model |
| `backend/models/prompt_template.py` | PromptTemplate ORM model |
| `backend/models/conversation.py` | Conversation ORM model |
| `backend/models/message.py` | Message ORM model |
| `backend/models/intent_trace.py` | IntentTrace ORM model |
| `backend/services/chatflow_service.py` | ChatFlow CRUD + connectivity test |
| `backend/services/category_service.py` | Category + skill mapping CRUD |
| `backend/services/model_service.py` | Model config CRUD |
| `backend/services/prompt_service.py` | Prompt CRUD with versioning |
| `backend/services/conversation_service.py` | Conversation lifecycle management |
| `backend/services/analytics_service.py` | Statistics queries |
| `backend/api/admin/chatflow.py` | Admin ChatFlow endpoints |
| `backend/api/admin/category.py` | Admin category endpoints |
| `backend/api/admin/model_config.py` | Admin model config endpoints |
| `backend/api/admin/prompt.py` | Admin prompt endpoints |
| `backend/api/admin/conversation_log.py` | Admin conversation log endpoints |
| `backend/api/admin/analytics.py` | Admin analytics endpoints |
| `backend/api/v1/conversation.py` | External conversation API (create/chat/reset/break) |
| `backend/api/v1/health.py` | Health check |
| `backend/api/middleware/logging.py` | Request logging middleware |
| `backend/api/deps.py` | FastAPI dependency injection (get_db, get_redis) |
| `backend/api/schemas/chatflow.py` | Pydantic schemas for ChatFlow API |
| `backend/api/schemas/category.py` | Pydantic schemas for Category API |
| `backend/api/schemas/model_config.py` | Pydantic schemas for ModelConfig API |
| `backend/api/schemas/prompt.py` | Pydantic schemas for Prompt API |
| `backend/api/schemas/conversation.py` | Pydantic schemas for Conversation API |
| `backend/engine/state.py` | IntentState TypedDict |
| `backend/engine/graph.py` | LangGraph graph assembly |
| `backend/engine/nodes/classify_l1.py` | Level-1 intent classification node |
| `backend/engine/nodes/expand_skills.py` | Skill expansion node |
| `backend/engine/nodes/classify_l2.py` | Level-2 intent selection node |
| `backend/engine/nodes/extract_rewrite.py` | Slot extraction + query rewrite node |
| `backend/engine/nodes/call_chatflow.py` | ChatFlow HTTP/SSE caller |
| `backend/engine/nodes/intent_switch.py` | Intent switch detection node |
| `backend/engine/nodes/process_result.py` | Result processing + state update |
| `alembic.ini` | Alembic config |
| `alembic/env.py` | Alembic migration environment |
| `tests/conftest.py` | pytest fixtures (test DB, test client, test redis) |
| `tests/test_models.py` | ORM model tests |
| `tests/test_services/test_chatflow_service.py` | ChatFlow service tests |
| `tests/test_services/test_category_service.py` | Category service tests |
| `tests/test_services/test_prompt_service.py` | Prompt service tests |
| `tests/test_api/test_chatflow_api.py` | ChatFlow API endpoint tests |
| `tests/test_api/test_category_api.py` | Category API endpoint tests |
| `tests/test_api/test_conversation_api.py` | Conversation API endpoint tests |
| `tests/test_engine/test_classify_l1.py` | L1 classification node tests |
| `tests/test_engine/test_classify_l2.py` | L2 classification node tests |
| `tests/test_engine/test_extract_rewrite.py` | Slot extraction tests |
| `tests/test_engine/test_intent_switch.py` | Intent switch detection tests |
| `tests/test_engine/test_graph.py` | Full graph integration test |
| `Dockerfile` | Backend container image |
| `docker-compose.yml` | Full stack compose |

---

## Phase 1: Project Scaffolding

### Task 1: Initialize project and install dependencies

**Files:**
- Create: `~/projects/ynet/intent-hub/pyproject.toml`
- Create: `~/projects/ynet/intent-hub/.gitignore`
- Create: `~/projects/ynet/intent-hub/.env.example`

- [ ] **Step 1: Create project directory and initialize git**

```bash
mkdir -p ~/projects/ynet/intent-hub
cd ~/projects/ynet/intent-hub
git init
```

- [ ] **Step 2: Create pyproject.toml**

```toml
[project]
name = "intent-hub"
version = "0.1.0"
description = "Intent Recognition Platform - Multi-agent routing with hierarchical skill disclosure"
requires-python = ">=3.11"
dependencies = [
    "fastapi>=0.110.0",
    "uvicorn[standard]>=0.27.0",
    "sse-starlette>=2.0.0",
    "sqlalchemy[asyncio]>=2.0.0",
    "aiomysql>=0.2.0",
    "alembic>=1.13.0",
    "redis[hiredis]>=5.0.0",
    "langgraph>=0.2.0",
    "langchain>=0.3.0",
    "langchain-openai>=0.2.0",
    "pydantic>=2.6.0",
    "pydantic-settings>=2.1.0",
    "cryptography>=42.0.0",
    "httpx>=0.27.0",
    "snowflake-id>=1.0.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=8.0.0",
    "pytest-asyncio>=0.23.0",
    "pytest-cov>=4.1.0",
    "httpx>=0.27.0",
    "aiosqlite>=0.20.0",
]

[tool.pytest.ini_options]
asyncio_mode = "auto"
testpaths = ["tests"]
```

- [ ] **Step 3: Create .gitignore**

```
__pycache__/
*.py[cod]
.env
*.db
.venv/
dist/
*.egg-info/
.pytest_cache/
.coverage
htmlcov/
```

- [ ] **Step 4: Create .env.example**

```env
DATABASE_URL=mysql+aiomysql://root:password@localhost:3306/intent_hub
REDIS_URL=redis://localhost:6379/0
SECRET_KEY=change-me-to-a-random-string
LLM_API_KEY=your-llm-api-key
LLM_API_ENDPOINT=https://api.openai.com/v1
LLM_MODEL_NAME=gpt-4o
```

- [ ] **Step 5: Create virtual environment and install**

```bash
cd ~/projects/ynet/intent-hub
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"
```

- [ ] **Step 6: Commit**

```bash
git add pyproject.toml .gitignore .env.example
git commit -m "chore: initialize intent-hub project with dependencies"
```

---

### Task 2: Core infrastructure (config, database, redis, security)

**Files:**
- Create: `backend/__init__.py`
- Create: `backend/core/__init__.py`
- Create: `backend/core/config.py`
- Create: `backend/core/database.py`
- Create: `backend/core/redis.py`
- Create: `backend/core/security.py`
- Create: `tests/__init__.py`
- Create: `tests/conftest.py`

- [ ] **Step 1: Create package structure**

```bash
mkdir -p backend/core backend/models backend/services backend/api/admin backend/api/v1 backend/api/schemas backend/api/middleware backend/engine/nodes tests/test_services tests/test_api tests/test_engine
touch backend/__init__.py backend/core/__init__.py backend/models/__init__.py backend/services/__init__.py backend/api/__init__.py backend/api/admin/__init__.py backend/api/v1/__init__.py backend/api/schemas/__init__.py backend/api/middleware/__init__.py backend/engine/__init__.py backend/engine/nodes/__init__.py tests/__init__.py tests/test_services/__init__.py tests/test_api/__init__.py tests/test_engine/__init__.py
```

- [ ] **Step 2: Write config.py**

```python
# backend/core/config.py
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    DATABASE_URL: str = "mysql+aiomysql://root:password@localhost:3306/intent_hub"
    DATABASE_URL_SYNC: str = "mysql+pymysql://root:password@localhost:3306/intent_hub"
    REDIS_URL: str = "redis://localhost:6379/0"
    SECRET_KEY: str = "change-me"

    # Default LLM
    LLM_API_KEY: str = ""
    LLM_API_ENDPOINT: str = "https://api.openai.com/v1"
    LLM_MODEL_NAME: str = "gpt-4o"

    # App
    APP_NAME: str = "Intent Hub"
    DEBUG: bool = False

    model_config = {"env_file": ".env", "env_file_encoding": "utf-8"}


settings = Settings()
```

- [ ] **Step 3: Write database.py**

```python
# backend/core/database.py
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from backend.core.config import settings

engine = create_async_engine(settings.DATABASE_URL, echo=settings.DEBUG, pool_pre_ping=True)
async_session_factory = async_sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)


async def get_db() -> AsyncSession:
    async with async_session_factory() as session:
        yield session
```

- [ ] **Step 4: Write redis.py**

```python
# backend/core/redis.py
import redis.asyncio as redis

from backend.core.config import settings

redis_pool = redis.ConnectionPool.from_url(settings.REDIS_URL)


async def get_redis() -> redis.Redis:
    return redis.Redis(connection_pool=redis_pool)
```

- [ ] **Step 5: Write security.py**

```python
# backend/core/security.py
from cryptography.fernet import Fernet

from backend.core.config import settings

_fernet = Fernet(Fernet.generate_key() if settings.SECRET_KEY == "change-me" else settings.SECRET_KEY.encode()[:44])


def encrypt_credential(plaintext: str) -> str:
    return _fernet.encrypt(plaintext.encode()).decode()


def decrypt_credential(ciphertext: str) -> str:
    return _fernet.decrypt(ciphertext.encode()).decode()
```

- [ ] **Step 6: Write conftest.py with test fixtures**

```python
# tests/conftest.py
import pytest
from httpx import ASGITransport, AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from backend.models.base import Base


@pytest.fixture
async def db_engine():
    engine = create_async_engine("sqlite+aiosqlite:///:memory:")
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    await engine.dispose()


@pytest.fixture
async def db_session(db_engine):
    session_factory = async_sessionmaker(db_engine, class_=AsyncSession, expire_on_commit=False)
    async with session_factory() as session:
        yield session


@pytest.fixture
async def client(db_session):
    from backend.api.deps import get_db_session
    from backend.main import app

    app.dependency_overrides[get_db_session] = lambda: db_session
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c
    app.dependency_overrides.clear()
```

- [ ] **Step 7: Commit**

```bash
git add backend/ tests/
git commit -m "feat(core): add config, database, redis, and security infrastructure"
```

---

### Task 3: FastAPI app entry point and health check

**Files:**
- Create: `backend/main.py`
- Create: `backend/api/v1/health.py`
- Create: `backend/api/deps.py`

- [ ] **Step 1: Write deps.py**

```python
# backend/api/deps.py
from sqlalchemy.ext.asyncio import AsyncSession

from backend.core.database import async_session_factory


async def get_db_session() -> AsyncSession:
    async with async_session_factory() as session:
        yield session
```

- [ ] **Step 2: Write health.py**

```python
# backend/api/v1/health.py
from fastapi import APIRouter

router = APIRouter()


@router.get("/health")
async def health():
    return {"status": "ok", "service": "intent-hub"}
```

- [ ] **Step 3: Write main.py**

```python
# backend/main.py
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from backend.api.v1 import health

app = FastAPI(title="Intent Hub", version="0.1.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(health.router, prefix="/api/v1", tags=["health"])
```

- [ ] **Step 4: Verify the app starts**

```bash
cd ~/projects/ynet/intent-hub
uvicorn backend.main:app --host 0.0.0.0 --port 8000 &
sleep 2
curl http://localhost:8000/api/v1/health
# Expected: {"status":"ok","service":"intent-hub"}
kill %1
```

- [ ] **Step 5: Commit**

```bash
git add backend/main.py backend/api/v1/health.py backend/api/deps.py
git commit -m "feat(api): add FastAPI app entry point and health check"
```

---

## Phase 2: Database Models & Migrations

### Task 4: SQLAlchemy ORM models

**Files:**
- Create: `backend/models/base.py`
- Create: `backend/models/chatflow_agent.py`
- Create: `backend/models/skill_category.py`
- Create: `backend/models/skill_mapping.py`
- Create: `backend/models/model_config.py`
- Create: `backend/models/prompt_template.py`
- Create: `backend/models/conversation.py`
- Create: `backend/models/message.py`
- Create: `backend/models/intent_trace.py`
- Test: `tests/test_models.py`

- [ ] **Step 1: Write the model test**

```python
# tests/test_models.py
import pytest
from sqlalchemy import select

from backend.models.chatflow_agent import ChatflowAgent
from backend.models.skill_category import SkillCategory
from backend.models.skill_mapping import SkillChatflowMapping
from backend.models.model_config import ModelConfig
from backend.models.prompt_template import PromptTemplate
from backend.models.conversation import Conversation
from backend.models.message import Message
from backend.models.intent_trace import IntentTrace


async def test_create_chatflow_agent(db_session):
    agent = ChatflowAgent(
        name="转账服务",
        description="处理用户的转账需求",
        endpoint="http://localhost:9000/chatflow/transfer",
        auth_type="bearer",
        auth_credential="encrypted-token",
        protocol="sse",
        status="active",
    )
    db_session.add(agent)
    await db_session.commit()

    result = await db_session.execute(select(ChatflowAgent).where(ChatflowAgent.name == "转账服务"))
    saved = result.scalar_one()
    assert saved.name == "转账服务"
    assert saved.protocol == "sse"
    assert saved.status == "active"


async def test_skill_category_with_mapping(db_session):
    category = SkillCategory(name="金融交易服务", description="处理转账、缴费、兑换等", status="active", sort_order=1)
    db_session.add(category)
    await db_session.commit()

    agent = ChatflowAgent(
        name="转账服务", description="转账", endpoint="http://x", auth_type="none", protocol="http_json", status="active"
    )
    db_session.add(agent)
    await db_session.commit()

    mapping = SkillChatflowMapping(
        category_id=category.id,
        chatflow_id=agent.id,
        skill_name="转账",
        skill_desc="用户发起转账请求",
        sort_order=1,
        extract_slots=[{"name": "payee", "type": "str", "desc": "收款人", "required": True}],
    )
    db_session.add(mapping)
    await db_session.commit()

    result = await db_session.execute(select(SkillChatflowMapping).where(SkillChatflowMapping.category_id == category.id))
    saved = result.scalar_one()
    assert saved.skill_name == "转账"
    assert saved.extract_slots[0]["name"] == "payee"


async def test_conversation_with_messages(db_session):
    conv = Conversation(external_id="conv-001", user_id="user-01", status="active")
    db_session.add(conv)
    await db_session.commit()

    msg = Message(
        conversation_id=conv.id, role="user", content="我想转5000给张三", source="user"
    )
    db_session.add(msg)
    await db_session.commit()

    result = await db_session.execute(select(Message).where(Message.conversation_id == conv.id))
    saved = result.scalar_one()
    assert saved.content == "我想转5000给张三"


async def test_prompt_template_versioning(db_session):
    prompt = PromptTemplate(
        name="一级意图识别",
        type="intent_l1",
        content="你是一个意图分类助手。{{categories}}",
        variables=["categories", "user_input", "chat_history"],
        is_active=True,
        version=1,
    )
    db_session.add(prompt)
    await db_session.commit()

    assert prompt.version == 1
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ~/projects/ynet/intent-hub
pytest tests/test_models.py -v
# Expected: FAIL - modules not found
```

- [ ] **Step 3: Write base.py**

```python
# backend/models/base.py
from datetime import datetime

from sqlalchemy import BigInteger, DateTime, func
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    pass


class TimestampMixin:
    created_at: Mapped[datetime] = mapped_column(DateTime, server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime, server_default=func.now(), onupdate=func.now())
```

- [ ] **Step 4: Write all 8 ORM models**

```python
# backend/models/chatflow_agent.py
from sqlalchemy import BigInteger, Enum, Integer, JSON, String, Text
from sqlalchemy.orm import Mapped, mapped_column

from backend.models.base import Base, TimestampMixin


class ChatflowAgent(Base, TimestampMixin):
    __tablename__ = "chatflow_agent"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    name: Mapped[str] = mapped_column(String(128))
    description: Mapped[str] = mapped_column(Text)
    endpoint: Mapped[str] = mapped_column(String(512))
    auth_type: Mapped[str] = mapped_column(Enum("none", "bearer", "api_key", name="auth_type_enum"), default="none")
    auth_credential: Mapped[str | None] = mapped_column(String(512), nullable=True)
    protocol: Mapped[str] = mapped_column(Enum("sse", "http_json", name="protocol_enum"), default="http_json")
    status: Mapped[str] = mapped_column(Enum("active", "inactive", "testing", name="cf_status_enum"), default="testing")
    timeout_ms: Mapped[int] = mapped_column(Integer, default=30000)
    metadata_: Mapped[dict | None] = mapped_column("metadata", JSON, nullable=True)
```

```python
# backend/models/skill_category.py
from sqlalchemy import BigInteger, Enum, Integer, String, Text
from sqlalchemy.orm import Mapped, mapped_column

from backend.models.base import Base, TimestampMixin


class SkillCategory(Base, TimestampMixin):
    __tablename__ = "skill_category"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    name: Mapped[str] = mapped_column(String(128))
    description: Mapped[str] = mapped_column(Text)
    icon: Mapped[str | None] = mapped_column(String(256), nullable=True)
    sort_order: Mapped[int] = mapped_column(Integer, default=0)
    status: Mapped[str] = mapped_column(Enum("active", "inactive", name="cat_status_enum"), default="active")
```

```python
# backend/models/skill_mapping.py
from sqlalchemy import BigInteger, ForeignKey, Integer, JSON, String, Text
from sqlalchemy.orm import Mapped, mapped_column

from backend.models.base import Base


class SkillChatflowMapping(Base):
    __tablename__ = "skill_chatflow_mapping"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    category_id: Mapped[int] = mapped_column(BigInteger, ForeignKey("skill_category.id"))
    chatflow_id: Mapped[int] = mapped_column(BigInteger, ForeignKey("chatflow_agent.id"))
    skill_name: Mapped[str] = mapped_column(String(128))
    skill_desc: Mapped[str] = mapped_column(Text)
    sort_order: Mapped[int] = mapped_column(Integer, default=0)
    extract_slots: Mapped[list | None] = mapped_column(JSON, nullable=True)
```

```python
# backend/models/model_config.py
from sqlalchemy import BigInteger, Boolean, Float, Integer, String
from sqlalchemy.orm import Mapped, mapped_column

from backend.models.base import Base, TimestampMixin


class ModelConfig(Base, TimestampMixin):
    __tablename__ = "model_config"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    name: Mapped[str] = mapped_column(String(128))
    provider: Mapped[str] = mapped_column(String(64))
    model_name: Mapped[str] = mapped_column(String(128))
    api_endpoint: Mapped[str] = mapped_column(String(512))
    api_key: Mapped[str] = mapped_column(String(512))
    temperature: Mapped[float] = mapped_column(Float, default=0.1)
    max_tokens: Mapped[int] = mapped_column(Integer, default=1024)
    is_default: Mapped[bool] = mapped_column(Boolean, default=False)
```

```python
# backend/models/prompt_template.py
from sqlalchemy import BigInteger, Boolean, Enum, Integer, JSON, String, Text
from sqlalchemy.orm import Mapped, mapped_column

from backend.models.base import Base, TimestampMixin


class PromptTemplate(Base, TimestampMixin):
    __tablename__ = "prompt_template"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    name: Mapped[str] = mapped_column(String(128))
    type: Mapped[str] = mapped_column(
        Enum("intent_l1", "intent_l2", "query_rewrite", "slot_extract", name="prompt_type_enum")
    )
    content: Mapped[str] = mapped_column(Text)
    variables: Mapped[list | None] = mapped_column(JSON, nullable=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    version: Mapped[int] = mapped_column(Integer, default=1)
```

```python
# backend/models/conversation.py
from sqlalchemy import BigInteger, Enum, JSON, String
from sqlalchemy.orm import Mapped, mapped_column

from backend.models.base import Base, TimestampMixin


class Conversation(Base, TimestampMixin):
    __tablename__ = "conversation"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    external_id: Mapped[str] = mapped_column(String(64), unique=True, index=True)
    user_id: Mapped[str] = mapped_column(String(128), index=True)
    status: Mapped[str] = mapped_column(Enum("active", "closed", name="conv_status_enum"), default="active")
    current_intent: Mapped[str | None] = mapped_column(String(128), nullable=True)
    current_chatflow_id: Mapped[int | None] = mapped_column(BigInteger, nullable=True)
    slot_state: Mapped[dict | None] = mapped_column(JSON, nullable=True)
```

```python
# backend/models/message.py
from sqlalchemy import BigInteger, Enum, ForeignKey, JSON, Text
from sqlalchemy.orm import Mapped, mapped_column

from backend.models.base import Base, TimestampMixin


class Message(Base, TimestampMixin):
    __tablename__ = "message"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    conversation_id: Mapped[int] = mapped_column(BigInteger, ForeignKey("conversation.id"), index=True)
    role: Mapped[str] = mapped_column(Enum("user", "assistant", "system", name="msg_role_enum"))
    content: Mapped[str] = mapped_column(Text)
    source: Mapped[str] = mapped_column(Enum("user", "platform", "chatflow", name="msg_source_enum"))
    chatflow_id: Mapped[int | None] = mapped_column(BigInteger, nullable=True)
    intent_snapshot: Mapped[dict | None] = mapped_column(JSON, nullable=True)
```

```python
# backend/models/intent_trace.py
from sqlalchemy import BigInteger, Enum, Float, ForeignKey, Integer, JSON, Text
from sqlalchemy.orm import Mapped, mapped_column

from backend.models.base import Base, TimestampMixin


class IntentTrace(Base, TimestampMixin):
    __tablename__ = "intent_trace"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    conversation_id: Mapped[int] = mapped_column(BigInteger, ForeignKey("conversation.id"), index=True)
    message_id: Mapped[int] = mapped_column(BigInteger, ForeignKey("message.id"))
    level: Mapped[str] = mapped_column(Enum("l1", "l2", name="intent_level_enum"))
    input_text: Mapped[str] = mapped_column(Text)
    category_id: Mapped[int | None] = mapped_column(BigInteger, nullable=True)
    chatflow_id: Mapped[int | None] = mapped_column(BigInteger, nullable=True)
    confidence: Mapped[float | None] = mapped_column(Float, nullable=True)
    reasoning: Mapped[str | None] = mapped_column(Text, nullable=True)
    slots_extracted: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    rewritten_query: Mapped[str | None] = mapped_column(Text, nullable=True)
    latency_ms: Mapped[int | None] = mapped_column(Integer, nullable=True)
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
pytest tests/test_models.py -v
# Expected: 4 passed
```

- [ ] **Step 6: Commit**

```bash
git add backend/models/
git commit -m "feat(models): add all 8 ORM models for intent hub"
```

---

### Task 5: Alembic migrations setup

**Files:**
- Create: `alembic.ini`
- Create: `alembic/env.py`
- Create: `alembic/versions/` (auto-generated)

- [ ] **Step 1: Initialize alembic**

```bash
cd ~/projects/ynet/intent-hub
alembic init alembic
```

- [ ] **Step 2: Edit alembic/env.py to use our models and async engine**

Replace `alembic/env.py` with:

```python
# alembic/env.py
from logging.config import fileConfig

from alembic import context
from sqlalchemy import create_engine

from backend.core.config import settings
from backend.models.base import Base
# Import all models so they're registered with Base.metadata
from backend.models import chatflow_agent, skill_category, skill_mapping  # noqa: F401
from backend.models import model_config, prompt_template  # noqa: F401
from backend.models import conversation, message, intent_trace  # noqa: F401

config = context.config
if config.config_file_name is not None:
    fileConfig(config.config_file_name)

target_metadata = Base.metadata


def run_migrations_offline():
    url = settings.DATABASE_URL_SYNC
    context.configure(url=url, target_metadata=target_metadata, literal_binds=True)
    with context.begin_transaction():
        context.run_migrations()


def run_migrations_online():
    connectable = create_engine(settings.DATABASE_URL_SYNC)
    with connectable.connect() as connection:
        context.configure(connection=connection, target_metadata=target_metadata)
        with context.begin_transaction():
            context.run_migrations()


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()
```

- [ ] **Step 3: Update alembic.ini sqlalchemy.url**

Set `sqlalchemy.url` in `alembic.ini` to empty (we use settings):

```ini
sqlalchemy.url =
```

- [ ] **Step 4: Generate initial migration**

```bash
alembic revision --autogenerate -m "initial tables"
```

- [ ] **Step 5: Commit**

```bash
git add alembic.ini alembic/
git commit -m "feat(db): add alembic migrations with initial schema"
```

---

## Phase 3: Management CRUD APIs

### Task 6: Pydantic schemas for all admin APIs

**Files:**
- Create: `backend/api/schemas/chatflow.py`
- Create: `backend/api/schemas/category.py`
- Create: `backend/api/schemas/model_config.py`
- Create: `backend/api/schemas/prompt.py`
- Create: `backend/api/schemas/conversation.py`
- Create: `backend/api/schemas/common.py`

- [ ] **Step 1: Write common schemas**

```python
# backend/api/schemas/common.py
from pydantic import BaseModel


class PageParams(BaseModel):
    page: int = 1
    page_size: int = 20


class PageResponse[T](BaseModel):
    total: int
    page: int
    page_size: int
    items: list[T]
```

- [ ] **Step 2: Write chatflow schemas**

```python
# backend/api/schemas/chatflow.py
from datetime import datetime

from pydantic import BaseModel


class ChatflowCreate(BaseModel):
    name: str
    description: str
    endpoint: str
    auth_type: str = "none"
    auth_credential: str | None = None
    protocol: str = "http_json"
    timeout_ms: int = 30000
    metadata: dict | None = None


class ChatflowUpdate(BaseModel):
    name: str | None = None
    description: str | None = None
    endpoint: str | None = None
    auth_type: str | None = None
    auth_credential: str | None = None
    protocol: str | None = None
    status: str | None = None
    timeout_ms: int | None = None
    metadata: dict | None = None


class ChatflowOut(BaseModel):
    id: int
    name: str
    description: str
    endpoint: str
    auth_type: str
    protocol: str
    status: str
    timeout_ms: int
    metadata: dict | None
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class ChatflowTestResult(BaseModel):
    success: bool
    latency_ms: int
    error: str | None = None
```

- [ ] **Step 3: Write category schemas**

```python
# backend/api/schemas/category.py
from datetime import datetime

from pydantic import BaseModel


class CategoryCreate(BaseModel):
    name: str
    description: str
    icon: str | None = None
    sort_order: int = 0


class CategoryUpdate(BaseModel):
    name: str | None = None
    description: str | None = None
    icon: str | None = None
    sort_order: int | None = None
    status: str | None = None


class CategoryOut(BaseModel):
    id: int
    name: str
    description: str
    icon: str | None
    sort_order: int
    status: str
    skill_count: int = 0
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class SkillCreate(BaseModel):
    chatflow_id: int
    skill_name: str
    skill_desc: str
    sort_order: int = 0
    extract_slots: list[dict] | None = None


class SkillUpdate(BaseModel):
    skill_name: str | None = None
    skill_desc: str | None = None
    sort_order: int | None = None
    extract_slots: list[dict] | None = None


class SkillOut(BaseModel):
    id: int
    category_id: int
    chatflow_id: int
    skill_name: str
    skill_desc: str
    sort_order: int
    extract_slots: list[dict] | None

    model_config = {"from_attributes": True}


class SortItem(BaseModel):
    id: int
    sort_order: int
```

- [ ] **Step 4: Write model_config schemas**

```python
# backend/api/schemas/model_config.py
from datetime import datetime

from pydantic import BaseModel


class ModelConfigCreate(BaseModel):
    name: str
    provider: str
    model_name: str
    api_endpoint: str
    api_key: str
    temperature: float = 0.1
    max_tokens: int = 1024


class ModelConfigUpdate(BaseModel):
    name: str | None = None
    provider: str | None = None
    model_name: str | None = None
    api_endpoint: str | None = None
    api_key: str | None = None
    temperature: float | None = None
    max_tokens: int | None = None


class ModelConfigOut(BaseModel):
    id: int
    name: str
    provider: str
    model_name: str
    api_endpoint: str
    temperature: float
    max_tokens: int
    is_default: bool
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}
```

- [ ] **Step 5: Write prompt schemas**

```python
# backend/api/schemas/prompt.py
from datetime import datetime

from pydantic import BaseModel


class PromptCreate(BaseModel):
    name: str
    type: str
    content: str
    variables: list[str] | None = None


class PromptUpdate(BaseModel):
    name: str | None = None
    content: str | None = None
    variables: list[str] | None = None
    is_active: bool | None = None


class PromptOut(BaseModel):
    id: int
    name: str
    type: str
    content: str
    variables: list[str] | None
    is_active: bool
    version: int
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class PromptVersionOut(BaseModel):
    version: int
    content: str
    created_at: datetime


class PromptRollback(BaseModel):
    target_version: int
```

- [ ] **Step 6: Write conversation schemas**

```python
# backend/api/schemas/conversation.py
from datetime import datetime

from pydantic import BaseModel


class ConversationCreate(BaseModel):
    user_id: str
    metadata: dict | None = None


class ConversationOut(BaseModel):
    conversation_id: str
    created_at: datetime


class ChatRequest(BaseModel):
    conversation_id: str
    query: str
    content_type: str = "text"


class MessageOut(BaseModel):
    id: int
    role: str
    content: str
    source: str
    chatflow_id: int | None
    intent_snapshot: dict | None
    created_at: datetime

    model_config = {"from_attributes": True}


class IntentTraceOut(BaseModel):
    id: int
    level: str
    input_text: str
    category_id: int | None
    chatflow_id: int | None
    confidence: float | None
    reasoning: str | None
    slots_extracted: dict | None
    rewritten_query: str | None
    latency_ms: int | None
    created_at: datetime

    model_config = {"from_attributes": True}
```

- [ ] **Step 7: Commit**

```bash
git add backend/api/schemas/
git commit -m "feat(schemas): add Pydantic schemas for all API endpoints"
```

---

### Task 7: ChatFlow service and admin API

**Files:**
- Create: `backend/services/chatflow_service.py`
- Create: `backend/api/admin/chatflow.py`
- Test: `tests/test_services/test_chatflow_service.py`
- Test: `tests/test_api/test_chatflow_api.py`

- [ ] **Step 1: Write service test**

```python
# tests/test_services/test_chatflow_service.py
import pytest

from backend.services.chatflow_service import ChatflowService


async def test_create_chatflow(db_session):
    svc = ChatflowService(db_session)
    agent = await svc.create(
        name="转账服务",
        description="处理用户的转账需求",
        endpoint="http://localhost:9000/transfer",
        auth_type="bearer",
        auth_credential="test-token",
        protocol="sse",
    )
    assert agent.id is not None
    assert agent.name == "转账服务"
    assert agent.status == "testing"


async def test_list_chatflows(db_session):
    svc = ChatflowService(db_session)
    await svc.create(name="A", description="a", endpoint="http://a", auth_type="none", protocol="http_json")
    await svc.create(name="B", description="b", endpoint="http://b", auth_type="none", protocol="http_json")

    items, total = await svc.list(page=1, page_size=10)
    assert total == 2
    assert len(items) == 2


async def test_update_chatflow(db_session):
    svc = ChatflowService(db_session)
    agent = await svc.create(name="Old", description="old", endpoint="http://x", auth_type="none", protocol="http_json")

    updated = await svc.update(agent.id, name="New", status="active")
    assert updated.name == "New"
    assert updated.status == "active"


async def test_delete_chatflow(db_session):
    svc = ChatflowService(db_session)
    agent = await svc.create(name="Del", description="del", endpoint="http://x", auth_type="none", protocol="http_json")

    await svc.delete(agent.id)
    result = await svc.get(agent.id)
    assert result.status == "inactive"
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
pytest tests/test_services/test_chatflow_service.py -v
# Expected: FAIL
```

- [ ] **Step 3: Write chatflow_service.py**

```python
# backend/services/chatflow_service.py
import time

import httpx
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.models.chatflow_agent import ChatflowAgent


class ChatflowService:
    def __init__(self, db: AsyncSession):
        self.db = db

    async def create(self, *, name: str, description: str, endpoint: str, auth_type: str, protocol: str,
                     auth_credential: str | None = None, timeout_ms: int = 30000, metadata: dict | None = None) -> ChatflowAgent:
        agent = ChatflowAgent(
            name=name, description=description, endpoint=endpoint, auth_type=auth_type,
            auth_credential=auth_credential, protocol=protocol, timeout_ms=timeout_ms, metadata_=metadata,
        )
        self.db.add(agent)
        await self.db.commit()
        await self.db.refresh(agent)
        return agent

    async def get(self, agent_id: int) -> ChatflowAgent | None:
        return await self.db.get(ChatflowAgent, agent_id)

    async def list(self, page: int = 1, page_size: int = 20, search: str | None = None) -> tuple[list[ChatflowAgent], int]:
        query = select(ChatflowAgent)
        count_query = select(func.count()).select_from(ChatflowAgent)

        if search:
            query = query.where(ChatflowAgent.name.contains(search))
            count_query = count_query.where(ChatflowAgent.name.contains(search))

        total = (await self.db.execute(count_query)).scalar() or 0
        query = query.order_by(ChatflowAgent.created_at.desc()).offset((page - 1) * page_size).limit(page_size)
        result = await self.db.execute(query)
        return list(result.scalars().all()), total

    async def update(self, agent_id: int, **kwargs) -> ChatflowAgent:
        agent = await self.db.get(ChatflowAgent, agent_id)
        for key, value in kwargs.items():
            if value is not None and hasattr(agent, key):
                setattr(agent, key, value)
        await self.db.commit()
        await self.db.refresh(agent)
        return agent

    async def delete(self, agent_id: int):
        agent = await self.db.get(ChatflowAgent, agent_id)
        agent.status = "inactive"
        await self.db.commit()

    async def test_connectivity(self, agent_id: int) -> dict:
        agent = await self.db.get(ChatflowAgent, agent_id)
        start = time.monotonic()
        try:
            async with httpx.AsyncClient(timeout=agent.timeout_ms / 1000) as client:
                headers = {}
                if agent.auth_type == "bearer" and agent.auth_credential:
                    headers["Authorization"] = f"Bearer {agent.auth_credential}"
                elif agent.auth_type == "api_key" and agent.auth_credential:
                    headers["X-API-Key"] = agent.auth_credential
                resp = await client.get(agent.endpoint, headers=headers)
                latency = int((time.monotonic() - start) * 1000)
                return {"success": resp.status_code < 500, "latency_ms": latency, "error": None}
        except Exception as e:
            latency = int((time.monotonic() - start) * 1000)
            return {"success": False, "latency_ms": latency, "error": str(e)}
```

- [ ] **Step 4: Run service tests**

```bash
pytest tests/test_services/test_chatflow_service.py -v
# Expected: 4 passed
```

- [ ] **Step 5: Write API endpoint test**

```python
# tests/test_api/test_chatflow_api.py
import pytest


async def test_chatflow_crud(client):
    # Create
    resp = await client.post("/api/admin/chatflows", json={
        "name": "转账服务", "description": "处理转账", "endpoint": "http://localhost:9000/transfer",
        "auth_type": "none", "protocol": "sse",
    })
    assert resp.status_code == 200
    data = resp.json()
    agent_id = data["id"]
    assert data["name"] == "转账服务"

    # List
    resp = await client.get("/api/admin/chatflows")
    assert resp.status_code == 200
    assert resp.json()["total"] == 1

    # Get
    resp = await client.get(f"/api/admin/chatflows/{agent_id}")
    assert resp.status_code == 200
    assert resp.json()["name"] == "转账服务"

    # Update
    resp = await client.put(f"/api/admin/chatflows/{agent_id}", json={"name": "转账服务v2"})
    assert resp.status_code == 200
    assert resp.json()["name"] == "转账服务v2"

    # Delete (soft)
    resp = await client.delete(f"/api/admin/chatflows/{agent_id}")
    assert resp.status_code == 200

    resp = await client.get(f"/api/admin/chatflows/{agent_id}")
    assert resp.json()["status"] == "inactive"
```

- [ ] **Step 6: Write chatflow admin API**

```python
# backend/api/admin/chatflow.py
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from backend.api.deps import get_db_session
from backend.api.schemas.chatflow import ChatflowCreate, ChatflowOut, ChatflowTestResult, ChatflowUpdate
from backend.api.schemas.common import PageResponse
from backend.services.chatflow_service import ChatflowService

router = APIRouter()


@router.post("/chatflows", response_model=ChatflowOut)
async def create_chatflow(body: ChatflowCreate, db: AsyncSession = Depends(get_db_session)):
    svc = ChatflowService(db)
    agent = await svc.create(**body.model_dump())
    return agent


@router.get("/chatflows", response_model=PageResponse[ChatflowOut])
async def list_chatflows(page: int = 1, page_size: int = 20, search: str | None = None,
                         db: AsyncSession = Depends(get_db_session)):
    svc = ChatflowService(db)
    items, total = await svc.list(page=page, page_size=page_size, search=search)
    return PageResponse(total=total, page=page, page_size=page_size, items=items)


@router.get("/chatflows/{agent_id}", response_model=ChatflowOut)
async def get_chatflow(agent_id: int, db: AsyncSession = Depends(get_db_session)):
    svc = ChatflowService(db)
    return await svc.get(agent_id)


@router.put("/chatflows/{agent_id}", response_model=ChatflowOut)
async def update_chatflow(agent_id: int, body: ChatflowUpdate, db: AsyncSession = Depends(get_db_session)):
    svc = ChatflowService(db)
    return await svc.update(agent_id, **body.model_dump(exclude_unset=True))


@router.delete("/chatflows/{agent_id}")
async def delete_chatflow(agent_id: int, db: AsyncSession = Depends(get_db_session)):
    svc = ChatflowService(db)
    await svc.delete(agent_id)
    return {"success": True}


@router.post("/chatflows/{agent_id}/test", response_model=ChatflowTestResult)
async def test_chatflow(agent_id: int, db: AsyncSession = Depends(get_db_session)):
    svc = ChatflowService(db)
    return await svc.test_connectivity(agent_id)
```

- [ ] **Step 7: Register router in main.py**

Add to `backend/main.py`:

```python
from backend.api.admin import chatflow as admin_chatflow

app.include_router(admin_chatflow.router, prefix="/api/admin", tags=["admin-chatflow"])
```

- [ ] **Step 8: Run API tests**

```bash
pytest tests/test_api/test_chatflow_api.py -v
# Expected: 1 passed
```

- [ ] **Step 9: Commit**

```bash
git add backend/services/chatflow_service.py backend/api/admin/chatflow.py tests/
git commit -m "feat(admin): add ChatFlow agent CRUD API with service layer"
```

---

### Task 8: Category and skill mapping service + API

**Files:**
- Create: `backend/services/category_service.py`
- Create: `backend/api/admin/category.py`
- Test: `tests/test_services/test_category_service.py`
- Test: `tests/test_api/test_category_api.py`

- [ ] **Step 1: Write service test**

```python
# tests/test_services/test_category_service.py
from backend.models.chatflow_agent import ChatflowAgent
from backend.services.category_service import CategoryService


async def test_create_category(db_session):
    svc = CategoryService(db_session)
    cat = await svc.create(name="金融交易服务", description="处理转账、缴费等", sort_order=1)
    assert cat.id is not None
    assert cat.name == "金融交易服务"


async def test_add_skill_mapping(db_session):
    svc = CategoryService(db_session)
    cat = await svc.create(name="金融交易", description="金融", sort_order=1)

    agent = ChatflowAgent(name="转账", description="转账", endpoint="http://x", auth_type="none", protocol="http_json", status="active")
    db_session.add(agent)
    await db_session.commit()

    skill = await svc.add_skill(
        category_id=cat.id, chatflow_id=agent.id,
        skill_name="转账", skill_desc="用户发起转账",
        extract_slots=[{"name": "amount", "type": "number", "desc": "金额", "required": True}],
    )
    assert skill.skill_name == "转账"

    skills = await svc.list_skills(cat.id)
    assert len(skills) == 1


async def test_batch_sort(db_session):
    svc = CategoryService(db_session)
    c1 = await svc.create(name="A", description="a", sort_order=1)
    c2 = await svc.create(name="B", description="b", sort_order=2)

    await svc.batch_sort([{"id": c1.id, "sort_order": 10}, {"id": c2.id, "sort_order": 5}])

    updated_c1 = await svc.get(c1.id)
    updated_c2 = await svc.get(c2.id)
    assert updated_c1.sort_order == 10
    assert updated_c2.sort_order == 5
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
pytest tests/test_services/test_category_service.py -v
# Expected: FAIL
```

- [ ] **Step 3: Write category_service.py**

```python
# backend/services/category_service.py
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.models.skill_category import SkillCategory
from backend.models.skill_mapping import SkillChatflowMapping


class CategoryService:
    def __init__(self, db: AsyncSession):
        self.db = db

    async def create(self, *, name: str, description: str, icon: str | None = None, sort_order: int = 0) -> SkillCategory:
        cat = SkillCategory(name=name, description=description, icon=icon, sort_order=sort_order)
        self.db.add(cat)
        await self.db.commit()
        await self.db.refresh(cat)
        return cat

    async def get(self, cat_id: int) -> SkillCategory | None:
        return await self.db.get(SkillCategory, cat_id)

    async def list(self) -> list[dict]:
        cats = (await self.db.execute(
            select(SkillCategory).where(SkillCategory.status == "active").order_by(SkillCategory.sort_order)
        )).scalars().all()

        result = []
        for cat in cats:
            count = (await self.db.execute(
                select(func.count()).select_from(SkillChatflowMapping).where(SkillChatflowMapping.category_id == cat.id)
            )).scalar() or 0
            d = {c.key: getattr(cat, c.key) for c in cat.__table__.columns}
            d["skill_count"] = count
            result.append(d)
        return result

    async def update(self, cat_id: int, **kwargs) -> SkillCategory:
        cat = await self.db.get(SkillCategory, cat_id)
        for key, value in kwargs.items():
            if value is not None and hasattr(cat, key):
                setattr(cat, key, value)
        await self.db.commit()
        await self.db.refresh(cat)
        return cat

    async def delete(self, cat_id: int):
        cat = await self.db.get(SkillCategory, cat_id)
        cat.status = "inactive"
        await self.db.commit()

    async def batch_sort(self, items: list[dict]):
        for item in items:
            cat = await self.db.get(SkillCategory, item["id"])
            if cat:
                cat.sort_order = item["sort_order"]
        await self.db.commit()

    # --- Skill Mapping ---

    async def add_skill(self, *, category_id: int, chatflow_id: int, skill_name: str, skill_desc: str,
                        sort_order: int = 0, extract_slots: list | None = None) -> SkillChatflowMapping:
        mapping = SkillChatflowMapping(
            category_id=category_id, chatflow_id=chatflow_id,
            skill_name=skill_name, skill_desc=skill_desc,
            sort_order=sort_order, extract_slots=extract_slots,
        )
        self.db.add(mapping)
        await self.db.commit()
        await self.db.refresh(mapping)
        return mapping

    async def list_skills(self, category_id: int) -> list[SkillChatflowMapping]:
        result = await self.db.execute(
            select(SkillChatflowMapping)
            .where(SkillChatflowMapping.category_id == category_id)
            .order_by(SkillChatflowMapping.sort_order)
        )
        return list(result.scalars().all())

    async def update_skill(self, skill_id: int, **kwargs) -> SkillChatflowMapping:
        mapping = await self.db.get(SkillChatflowMapping, skill_id)
        for key, value in kwargs.items():
            if value is not None and hasattr(mapping, key):
                setattr(mapping, key, value)
        await self.db.commit()
        await self.db.refresh(mapping)
        return mapping

    async def delete_skill(self, skill_id: int):
        mapping = await self.db.get(SkillChatflowMapping, skill_id)
        await self.db.delete(mapping)
        await self.db.commit()
```

- [ ] **Step 4: Run service tests**

```bash
pytest tests/test_services/test_category_service.py -v
# Expected: 3 passed
```

- [ ] **Step 5: Write category admin API**

```python
# backend/api/admin/category.py
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from backend.api.deps import get_db_session
from backend.api.schemas.category import (CategoryCreate, CategoryOut, CategoryUpdate,
                                          SkillCreate, SkillOut, SkillUpdate, SortItem)
from backend.services.category_service import CategoryService

router = APIRouter()


@router.post("/categories", response_model=CategoryOut)
async def create_category(body: CategoryCreate, db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    cat = await svc.create(**body.model_dump())
    return CategoryOut(**{c.key: getattr(cat, c.key) for c in cat.__table__.columns}, skill_count=0)


@router.get("/categories")
async def list_categories(db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    return await svc.list()


@router.put("/categories/{cat_id}", response_model=CategoryOut)
async def update_category(cat_id: int, body: CategoryUpdate, db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    cat = await svc.update(cat_id, **body.model_dump(exclude_unset=True))
    return CategoryOut(**{c.key: getattr(cat, c.key) for c in cat.__table__.columns}, skill_count=0)


@router.delete("/categories/{cat_id}")
async def delete_category(cat_id: int, db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    await svc.delete(cat_id)
    return {"success": True}


@router.put("/categories/sort")
async def sort_categories(items: list[SortItem], db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    await svc.batch_sort([i.model_dump() for i in items])
    return {"success": True}


@router.post("/categories/{cat_id}/skills", response_model=SkillOut)
async def add_skill(cat_id: int, body: SkillCreate, db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    return await svc.add_skill(category_id=cat_id, **body.model_dump())


@router.get("/categories/{cat_id}/skills", response_model=list[SkillOut])
async def list_skills(cat_id: int, db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    return await svc.list_skills(cat_id)


@router.put("/categories/{cat_id}/skills/{skill_id}", response_model=SkillOut)
async def update_skill(cat_id: int, skill_id: int, body: SkillUpdate, db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    return await svc.update_skill(skill_id, **body.model_dump(exclude_unset=True))


@router.delete("/categories/{cat_id}/skills/{skill_id}")
async def delete_skill(cat_id: int, skill_id: int, db: AsyncSession = Depends(get_db_session)):
    svc = CategoryService(db)
    await svc.delete_skill(skill_id)
    return {"success": True}
```

- [ ] **Step 6: Register router in main.py**

Add to `backend/main.py`:

```python
from backend.api.admin import category as admin_category

app.include_router(admin_category.router, prefix="/api/admin", tags=["admin-category"])
```

- [ ] **Step 7: Run all tests**

```bash
pytest tests/ -v
# Expected: all passed
```

- [ ] **Step 8: Commit**

```bash
git add backend/services/category_service.py backend/api/admin/category.py tests/
git commit -m "feat(admin): add skill category and mapping CRUD API"
```

---

### Task 9: Model config and prompt template APIs

**Files:**
- Create: `backend/services/model_service.py`
- Create: `backend/services/prompt_service.py`
- Create: `backend/api/admin/model_config.py`
- Create: `backend/api/admin/prompt.py`
- Test: `tests/test_services/test_prompt_service.py`

- [ ] **Step 1: Write prompt service test (versioning is the key logic)**

```python
# tests/test_services/test_prompt_service.py
from backend.services.prompt_service import PromptService


async def test_create_prompt(db_session):
    svc = PromptService(db_session)
    prompt = await svc.create(name="L1 Intent", type="intent_l1", content="Classify: {{categories}}", variables=["categories"])
    assert prompt.version == 1


async def test_update_increments_version(db_session):
    svc = PromptService(db_session)
    prompt = await svc.create(name="L1", type="intent_l1", content="v1 content")

    updated = await svc.update(prompt.id, content="v2 content")
    assert updated.version == 2
    assert updated.content == "v2 content"

    versions = await svc.get_versions(prompt.id)
    assert len(versions) == 2


async def test_rollback(db_session):
    svc = PromptService(db_session)
    prompt = await svc.create(name="L1", type="intent_l1", content="v1")
    await svc.update(prompt.id, content="v2")
    await svc.update(prompt.id, content="v3")

    rolled = await svc.rollback(prompt.id, target_version=1)
    assert rolled.content == "v1"
    assert rolled.version == 4  # New version with old content
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
pytest tests/test_services/test_prompt_service.py -v
# Expected: FAIL
```

- [ ] **Step 3: Write model_service.py**

```python
# backend/services/model_service.py
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.models.model_config import ModelConfig


class ModelService:
    def __init__(self, db: AsyncSession):
        self.db = db

    async def create(self, **kwargs) -> ModelConfig:
        config = ModelConfig(**kwargs)
        self.db.add(config)
        await self.db.commit()
        await self.db.refresh(config)
        return config

    async def get(self, config_id: int) -> ModelConfig | None:
        return await self.db.get(ModelConfig, config_id)

    async def get_default(self) -> ModelConfig | None:
        result = await self.db.execute(select(ModelConfig).where(ModelConfig.is_default == True))
        return result.scalar_one_or_none()

    async def list(self) -> list[ModelConfig]:
        result = await self.db.execute(select(ModelConfig).order_by(ModelConfig.created_at.desc()))
        return list(result.scalars().all())

    async def update(self, config_id: int, **kwargs) -> ModelConfig:
        config = await self.db.get(ModelConfig, config_id)
        for key, value in kwargs.items():
            if value is not None and hasattr(config, key):
                setattr(config, key, value)
        await self.db.commit()
        await self.db.refresh(config)
        return config

    async def set_default(self, config_id: int) -> ModelConfig:
        # Unset current default
        current = await self.get_default()
        if current:
            current.is_default = False

        config = await self.db.get(ModelConfig, config_id)
        config.is_default = True
        await self.db.commit()
        await self.db.refresh(config)
        return config
```

- [ ] **Step 4: Write prompt_service.py with versioning**

```python
# backend/services/prompt_service.py
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.models.prompt_template import PromptTemplate


class PromptService:
    def __init__(self, db: AsyncSession):
        self.db = db

    async def create(self, *, name: str, type: str, content: str, variables: list | None = None) -> PromptTemplate:
        prompt = PromptTemplate(name=name, type=type, content=content, variables=variables, version=1)
        self.db.add(prompt)
        await self.db.commit()
        await self.db.refresh(prompt)

        # Save version history as a separate record (same name+type, different version)
        return prompt

    async def get(self, prompt_id: int) -> PromptTemplate | None:
        return await self.db.get(PromptTemplate, prompt_id)

    async def get_active_by_type(self, prompt_type: str) -> PromptTemplate | None:
        result = await self.db.execute(
            select(PromptTemplate)
            .where(PromptTemplate.type == prompt_type, PromptTemplate.is_active == True)
            .order_by(PromptTemplate.version.desc())
            .limit(1)
        )
        return result.scalar_one_or_none()

    async def list(self, prompt_type: str | None = None) -> list[PromptTemplate]:
        query = select(PromptTemplate).where(PromptTemplate.is_active == True)
        if prompt_type:
            query = query.where(PromptTemplate.type == prompt_type)
        # Group by name, only show latest version
        query = query.order_by(PromptTemplate.name, PromptTemplate.version.desc())
        result = await self.db.execute(query)
        seen_names = set()
        prompts = []
        for p in result.scalars().all():
            if p.name not in seen_names:
                seen_names.add(p.name)
                prompts.append(p)
        return prompts

    async def update(self, prompt_id: int, **kwargs) -> PromptTemplate:
        old = await self.db.get(PromptTemplate, prompt_id)
        # Create new version
        new_prompt = PromptTemplate(
            name=old.name,
            type=old.type,
            content=kwargs.get("content", old.content),
            variables=kwargs.get("variables", old.variables),
            is_active=kwargs.get("is_active", old.is_active),
            version=old.version + 1,
        )
        old.is_active = False
        self.db.add(new_prompt)
        await self.db.commit()
        await self.db.refresh(new_prompt)
        return new_prompt

    async def get_versions(self, prompt_id: int) -> list[PromptTemplate]:
        prompt = await self.db.get(PromptTemplate, prompt_id)
        result = await self.db.execute(
            select(PromptTemplate)
            .where(PromptTemplate.name == prompt.name, PromptTemplate.type == prompt.type)
            .order_by(PromptTemplate.version.desc())
        )
        return list(result.scalars().all())

    async def rollback(self, prompt_id: int, target_version: int) -> PromptTemplate:
        prompt = await self.db.get(PromptTemplate, prompt_id)
        # Find target version
        result = await self.db.execute(
            select(PromptTemplate).where(
                PromptTemplate.name == prompt.name,
                PromptTemplate.type == prompt.type,
                PromptTemplate.version == target_version,
            )
        )
        target = result.scalar_one()

        # Get current max version
        max_result = await self.db.execute(
            select(PromptTemplate)
            .where(PromptTemplate.name == prompt.name, PromptTemplate.type == prompt.type)
            .order_by(PromptTemplate.version.desc())
            .limit(1)
        )
        current_max = max_result.scalar_one()
        current_max.is_active = False

        # Create new version with old content
        rolled = PromptTemplate(
            name=target.name, type=target.type, content=target.content,
            variables=target.variables, is_active=True, version=current_max.version + 1,
        )
        self.db.add(rolled)
        await self.db.commit()
        await self.db.refresh(rolled)
        return rolled
```

- [ ] **Step 5: Run prompt service tests**

```bash
pytest tests/test_services/test_prompt_service.py -v
# Expected: 3 passed
```

- [ ] **Step 6: Write model_config admin API**

```python
# backend/api/admin/model_config.py
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from backend.api.deps import get_db_session
from backend.api.schemas.model_config import ModelConfigCreate, ModelConfigOut, ModelConfigUpdate
from backend.services.model_service import ModelService

router = APIRouter()


@router.post("/models", response_model=ModelConfigOut)
async def create_model(body: ModelConfigCreate, db: AsyncSession = Depends(get_db_session)):
    svc = ModelService(db)
    return await svc.create(**body.model_dump())


@router.get("/models", response_model=list[ModelConfigOut])
async def list_models(db: AsyncSession = Depends(get_db_session)):
    svc = ModelService(db)
    return await svc.list()


@router.put("/models/{config_id}", response_model=ModelConfigOut)
async def update_model(config_id: int, body: ModelConfigUpdate, db: AsyncSession = Depends(get_db_session)):
    svc = ModelService(db)
    return await svc.update(config_id, **body.model_dump(exclude_unset=True))


@router.put("/models/{config_id}/default", response_model=ModelConfigOut)
async def set_default_model(config_id: int, db: AsyncSession = Depends(get_db_session)):
    svc = ModelService(db)
    return await svc.set_default(config_id)
```

- [ ] **Step 7: Write prompt admin API**

```python
# backend/api/admin/prompt.py
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from backend.api.deps import get_db_session
from backend.api.schemas.prompt import PromptCreate, PromptOut, PromptRollback, PromptUpdate
from backend.services.prompt_service import PromptService

router = APIRouter()


@router.post("/prompts", response_model=PromptOut)
async def create_prompt(body: PromptCreate, db: AsyncSession = Depends(get_db_session)):
    svc = PromptService(db)
    return await svc.create(**body.model_dump())


@router.get("/prompts", response_model=list[PromptOut])
async def list_prompts(type: str | None = None, db: AsyncSession = Depends(get_db_session)):
    svc = PromptService(db)
    return await svc.list(prompt_type=type)


@router.put("/prompts/{prompt_id}", response_model=PromptOut)
async def update_prompt(prompt_id: int, body: PromptUpdate, db: AsyncSession = Depends(get_db_session)):
    svc = PromptService(db)
    return await svc.update(prompt_id, **body.model_dump(exclude_unset=True))


@router.get("/prompts/{prompt_id}/versions", response_model=list[PromptOut])
async def get_versions(prompt_id: int, db: AsyncSession = Depends(get_db_session)):
    svc = PromptService(db)
    return await svc.get_versions(prompt_id)


@router.post("/prompts/{prompt_id}/rollback", response_model=PromptOut)
async def rollback_prompt(prompt_id: int, body: PromptRollback, db: AsyncSession = Depends(get_db_session)):
    svc = PromptService(db)
    return await svc.rollback(prompt_id, target_version=body.target_version)
```

- [ ] **Step 8: Register routers in main.py**

Add to `backend/main.py`:

```python
from backend.api.admin import model_config as admin_model
from backend.api.admin import prompt as admin_prompt

app.include_router(admin_model.router, prefix="/api/admin", tags=["admin-model"])
app.include_router(admin_prompt.router, prefix="/api/admin", tags=["admin-prompt"])
```

- [ ] **Step 9: Run all tests**

```bash
pytest tests/ -v
# Expected: all passed
```

- [ ] **Step 10: Commit**

```bash
git add backend/services/ backend/api/admin/ tests/
git commit -m "feat(admin): add model config and prompt template APIs with versioning"
```

---

## Phase 4: LangGraph Intent Engine

### Task 10: IntentState definition and graph skeleton

**Files:**
- Create: `backend/engine/state.py`
- Create: `backend/engine/graph.py`

- [ ] **Step 1: Write state.py**

```python
# backend/engine/state.py
from typing import TypedDict


class IntentState(TypedDict, total=False):
    # Input
    user_input: str
    conversation_id: str
    chat_history: list[dict]           # [{role, content, source}]

    # L1 intent
    l1_category_id: int | None
    l1_category_name: str | None
    l1_confidence: float | None

    # L2 intent
    available_skills: list[dict]       # Skills under selected category
    l2_chatflow_id: int | None
    l2_skill_name: str | None
    l2_confidence: float | None

    # Slots & Query
    slot_definitions: list[dict]
    extracted_slots: dict
    accumulated_slots: dict
    rewritten_query: str | None

    # ChatFlow interaction
    chatflow_endpoint: str | None
    chatflow_protocol: str | None
    chatflow_auth_type: str | None
    chatflow_auth_credential: str | None
    chatflow_response: str | None

    # Flow control
    current_phase: str                 # idle | classifying | in_chatflow
    intent_switched: bool
    error: str | None

    # SSE callback (not serialized, set at runtime)
    sse_send: object | None
```

- [ ] **Step 2: Write graph.py skeleton (nodes will be added in subsequent tasks)**

```python
# backend/engine/graph.py
from langgraph.graph import END, StateGraph

from backend.engine.state import IntentState


def route_entry(state: IntentState) -> str:
    if state.get("current_phase") == "in_chatflow":
        return "intent_switch_check"
    return "classify_l1"


def build_graph() -> StateGraph:
    graph = StateGraph(IntentState)
    # Nodes will be registered here as they are implemented
    return graph
```

- [ ] **Step 3: Commit**

```bash
git add backend/engine/
git commit -m "feat(engine): add IntentState definition and graph skeleton"
```

---

### Task 11: L1 classification node

**Files:**
- Create: `backend/engine/nodes/classify_l1.py`
- Test: `tests/test_engine/test_classify_l1.py`

- [ ] **Step 1: Write test**

```python
# tests/test_engine/test_classify_l1.py
from unittest.mock import AsyncMock, patch

from backend.engine.nodes.classify_l1 import classify_l1
from backend.engine.state import IntentState


async def test_classify_l1_selects_category():
    state: IntentState = {
        "user_input": "我想转5000给张三",
        "conversation_id": "test-conv",
        "chat_history": [],
        "current_phase": "idle",
        "accumulated_slots": {},
    }

    mock_categories = [
        {"id": 1, "name": "金融交易服务", "description": "处理转账、缴费、兑换等金融交易"},
        {"id": 2, "name": "账户管理服务", "description": "查询余额、修改账户设置等"},
    ]

    mock_llm_response = {"category_id": 1, "category_name": "金融交易服务", "confidence": 0.96}

    with patch("backend.engine.nodes.classify_l1.load_active_categories", return_value=mock_categories), \
         patch("backend.engine.nodes.classify_l1.call_llm_for_classification", return_value=mock_llm_response):
        result = await classify_l1(state)

    assert result["l1_category_id"] == 1
    assert result["l1_category_name"] == "金融交易服务"
    assert result["l1_confidence"] == 0.96
```

- [ ] **Step 2: Run test to verify it fails**

```bash
pytest tests/test_engine/test_classify_l1.py -v
# Expected: FAIL
```

- [ ] **Step 3: Write classify_l1.py**

```python
# backend/engine/nodes/classify_l1.py
import json
import time

from langchain_openai import ChatOpenAI
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.core.database import async_session_factory
from backend.engine.state import IntentState
from backend.models.model_config import ModelConfig
from backend.models.skill_category import SkillCategory


async def load_active_categories(db: AsyncSession | None = None) -> list[dict]:
    if db is None:
        async with async_session_factory() as db:
            return await load_active_categories(db)

    result = await db.execute(
        select(SkillCategory).where(SkillCategory.status == "active").order_by(SkillCategory.sort_order)
    )
    return [{"id": c.id, "name": c.name, "description": c.description} for c in result.scalars().all()]


async def call_llm_for_classification(user_input: str, categories: list[dict], chat_history: list[dict],
                                       llm: ChatOpenAI | None = None) -> dict:
    categories_text = "\n".join(f"{i+1}. {c['name']}: {c['description']}" for i, c in enumerate(categories))

    history_text = ""
    if chat_history:
        history_text = "\n".join(f"{m['role']}: {m['content']}" for m in chat_history[-10:])

    prompt = f"""你是一个意图分类助手。根据用户输入和对话历史，从以下服务类别中选择最匹配的一项。

服务类别:
{categories_text}

对话历史:
{history_text or '(无)'}

用户输入: {user_input}

请返回JSON格式:
{{"category_id": <类别序号对应的实际ID>, "category_name": "<类别名>", "confidence": <0-1的置信度>, "reason": "<简要理由>"}}

只返回JSON，不要其他内容。"""

    if llm is None:
        async with async_session_factory() as db:
            config = (await db.execute(select(ModelConfig).where(ModelConfig.is_default == True))).scalar_one_or_none()
            if config:
                llm = ChatOpenAI(model=config.model_name, api_key=config.api_key,
                                 base_url=config.api_endpoint, temperature=config.temperature)
            else:
                raise ValueError("No default model configured")

    response = await llm.ainvoke(prompt)
    return json.loads(response.content)


async def classify_l1(state: IntentState) -> dict:
    start = time.monotonic()

    categories = await load_active_categories()
    if not categories:
        return {"error": "No active categories configured", "l1_category_id": None}

    result = await call_llm_for_classification(
        user_input=state["user_input"],
        categories=categories,
        chat_history=state.get("chat_history", []),
    )

    latency = int((time.monotonic() - start) * 1000)

    return {
        "l1_category_id": result["category_id"],
        "l1_category_name": result["category_name"],
        "l1_confidence": result.get("confidence", 0.0),
        "current_phase": "classifying",
    }
```

- [ ] **Step 4: Run test**

```bash
pytest tests/test_engine/test_classify_l1.py -v
# Expected: 1 passed
```

- [ ] **Step 5: Commit**

```bash
git add backend/engine/nodes/classify_l1.py tests/test_engine/test_classify_l1.py
git commit -m "feat(engine): add L1 intent classification node"
```

---

### Task 12: Skill expansion + L2 classification nodes

**Files:**
- Create: `backend/engine/nodes/expand_skills.py`
- Create: `backend/engine/nodes/classify_l2.py`
- Test: `tests/test_engine/test_classify_l2.py`

- [ ] **Step 1: Write test**

```python
# tests/test_engine/test_classify_l2.py
from unittest.mock import AsyncMock, patch

from backend.engine.nodes.classify_l2 import classify_l2
from backend.engine.nodes.expand_skills import expand_skills
from backend.engine.state import IntentState


async def test_expand_skills_loads_from_db():
    state: IntentState = {"l1_category_id": 1}
    mock_skills = [
        {"id": 1, "chatflow_id": 101, "skill_name": "转账", "skill_desc": "用户发起转账", "extract_slots": [{"name": "amount"}]},
        {"id": 2, "chatflow_id": 102, "skill_name": "缴费", "skill_desc": "缴纳费用", "extract_slots": []},
    ]

    with patch("backend.engine.nodes.expand_skills.load_skills_for_category", return_value=mock_skills):
        result = await expand_skills(state)

    assert len(result["available_skills"]) == 2
    assert result["available_skills"][0]["skill_name"] == "转账"


async def test_classify_l2_selects_chatflow():
    state: IntentState = {
        "user_input": "我想转5000给张三",
        "chat_history": [],
        "available_skills": [
            {"id": 1, "chatflow_id": 101, "skill_name": "转账", "skill_desc": "用户发起转账",
             "extract_slots": [{"name": "payee"}, {"name": "amount"}],
             "endpoint": "http://x", "protocol": "sse", "auth_type": "none", "auth_credential": None},
            {"id": 2, "chatflow_id": 102, "skill_name": "缴费", "skill_desc": "缴纳费用",
             "extract_slots": [], "endpoint": "http://y", "protocol": "http_json", "auth_type": "none", "auth_credential": None},
        ],
    }

    mock_result = {"chatflow_id": 101, "skill_name": "转账", "confidence": 0.93}

    with patch("backend.engine.nodes.classify_l2.call_llm_for_l2", return_value=mock_result):
        result = await classify_l2(state)

    assert result["l2_chatflow_id"] == 101
    assert result["l2_skill_name"] == "转账"
    assert result["chatflow_endpoint"] == "http://x"
    assert result["slot_definitions"][0]["name"] == "payee"
```

- [ ] **Step 2: Run test to verify it fails**

```bash
pytest tests/test_engine/test_classify_l2.py -v
# Expected: FAIL
```

- [ ] **Step 3: Write expand_skills.py**

```python
# backend/engine/nodes/expand_skills.py
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.core.database import async_session_factory
from backend.engine.state import IntentState
from backend.models.chatflow_agent import ChatflowAgent
from backend.models.skill_mapping import SkillChatflowMapping


async def load_skills_for_category(category_id: int, db: AsyncSession | None = None) -> list[dict]:
    if db is None:
        async with async_session_factory() as db:
            return await load_skills_for_category(category_id, db)

    result = await db.execute(
        select(SkillChatflowMapping, ChatflowAgent)
        .join(ChatflowAgent, SkillChatflowMapping.chatflow_id == ChatflowAgent.id)
        .where(SkillChatflowMapping.category_id == category_id, ChatflowAgent.status == "active")
        .order_by(SkillChatflowMapping.sort_order)
    )

    skills = []
    for mapping, agent in result.all():
        skills.append({
            "id": mapping.id,
            "chatflow_id": agent.id,
            "skill_name": mapping.skill_name,
            "skill_desc": mapping.skill_desc,
            "extract_slots": mapping.extract_slots or [],
            "endpoint": agent.endpoint,
            "protocol": agent.protocol,
            "auth_type": agent.auth_type,
            "auth_credential": agent.auth_credential,
        })
    return skills


async def expand_skills(state: IntentState) -> dict:
    category_id = state["l1_category_id"]
    skills = await load_skills_for_category(category_id)
    return {"available_skills": skills}
```

- [ ] **Step 4: Write classify_l2.py**

```python
# backend/engine/nodes/classify_l2.py
import json
import time

from langchain_openai import ChatOpenAI
from sqlalchemy import select

from backend.core.database import async_session_factory
from backend.engine.state import IntentState
from backend.models.model_config import ModelConfig


async def call_llm_for_l2(user_input: str, skills: list[dict], chat_history: list[dict],
                           llm: ChatOpenAI | None = None) -> dict:
    skills_text = "\n".join(f"- {s['skill_name']} (ID:{s['chatflow_id']}): {s['skill_desc']}" for s in skills)

    prompt = f"""你是一个意图选择助手。用户已被分类到某个服务类别下，现在需要选择具体的技能。

可用技能:
{skills_text}

用户输入: {user_input}

请返回JSON格式:
{{"chatflow_id": <技能对应的ID>, "skill_name": "<技能名>", "confidence": <0-1>, "reason": "<理由>"}}

只返回JSON。"""

    if llm is None:
        async with async_session_factory() as db:
            config = (await db.execute(select(ModelConfig).where(ModelConfig.is_default == True))).scalar_one_or_none()
            if config:
                llm = ChatOpenAI(model=config.model_name, api_key=config.api_key,
                                 base_url=config.api_endpoint, temperature=config.temperature)
            else:
                raise ValueError("No default model configured")

    response = await llm.ainvoke(prompt)
    return json.loads(response.content)


async def classify_l2(state: IntentState) -> dict:
    skills = state["available_skills"]

    if len(skills) == 1:
        # Only one skill, skip LLM call
        s = skills[0]
        return {
            "l2_chatflow_id": s["chatflow_id"],
            "l2_skill_name": s["skill_name"],
            "l2_confidence": 1.0,
            "chatflow_endpoint": s["endpoint"],
            "chatflow_protocol": s["protocol"],
            "chatflow_auth_type": s["auth_type"],
            "chatflow_auth_credential": s["auth_credential"],
            "slot_definitions": s["extract_slots"],
        }

    result = await call_llm_for_l2(
        user_input=state["user_input"],
        skills=skills,
        chat_history=state.get("chat_history", []),
    )

    selected = next(s for s in skills if s["chatflow_id"] == result["chatflow_id"])

    return {
        "l2_chatflow_id": result["chatflow_id"],
        "l2_skill_name": result["skill_name"],
        "l2_confidence": result.get("confidence", 0.0),
        "chatflow_endpoint": selected["endpoint"],
        "chatflow_protocol": selected["protocol"],
        "chatflow_auth_type": selected["auth_type"],
        "chatflow_auth_credential": selected["auth_credential"],
        "slot_definitions": selected["extract_slots"],
    }
```

- [ ] **Step 5: Run tests**

```bash
pytest tests/test_engine/test_classify_l2.py -v
# Expected: 2 passed
```

- [ ] **Step 6: Commit**

```bash
git add backend/engine/nodes/expand_skills.py backend/engine/nodes/classify_l2.py tests/test_engine/test_classify_l2.py
git commit -m "feat(engine): add skill expansion and L2 classification nodes"
```

---

### Task 13: Slot extraction + query rewrite node

**Files:**
- Create: `backend/engine/nodes/extract_rewrite.py`
- Test: `tests/test_engine/test_extract_rewrite.py`

- [ ] **Step 1: Write test**

```python
# tests/test_engine/test_extract_rewrite.py
from unittest.mock import patch

from backend.engine.nodes.extract_rewrite import extract_and_rewrite
from backend.engine.state import IntentState


async def test_extract_slots_and_rewrite():
    state: IntentState = {
        "user_input": "我想转5000给张三",
        "chat_history": [],
        "l2_skill_name": "转账",
        "slot_definitions": [
            {"name": "payee", "type": "str", "desc": "收款人", "required": True},
            {"name": "amount", "type": "number", "desc": "转账金额", "required": True},
            {"name": "from_account", "type": "str", "desc": "付款账户", "required": False},
        ],
        "accumulated_slots": {},
    }

    mock_result = {
        "slots": {"payee": "张三", "amount": 5000, "from_account": None},
        "query": "用户要转账5000元给张三，付款账户待确认，请协助完成转账流程",
    }

    with patch("backend.engine.nodes.extract_rewrite.call_llm_for_extraction", return_value=mock_result):
        result = await extract_and_rewrite(state)

    assert result["extracted_slots"]["payee"] == "张三"
    assert result["extracted_slots"]["amount"] == 5000
    assert result["accumulated_slots"]["payee"] == "张三"
    assert "转账" in result["rewritten_query"]
```

- [ ] **Step 2: Run test to verify it fails**

```bash
pytest tests/test_engine/test_extract_rewrite.py -v
# Expected: FAIL
```

- [ ] **Step 3: Write extract_rewrite.py**

```python
# backend/engine/nodes/extract_rewrite.py
import json

from langchain_openai import ChatOpenAI
from sqlalchemy import select

from backend.core.database import async_session_factory
from backend.engine.state import IntentState
from backend.models.model_config import ModelConfig


async def call_llm_for_extraction(user_input: str, chat_history: list[dict], skill_name: str,
                                   slot_definitions: list[dict], llm: ChatOpenAI | None = None) -> dict:
    slots_text = "\n".join(
        f"- {s['name']} ({s['type']}): {s['desc']} {'[必填]' if s.get('required') else '[可选]'}"
        for s in slot_definitions
    )

    history_text = "\n".join(f"{m['role']}: {m['content']}" for m in chat_history[-10:]) if chat_history else "(无)"

    prompt = f"""你是一个信息提取助手。用户的意图是"{skill_name}"。

请从用户输入和对话历史中提取以下信息，并将用户意图重组为一句完整的query。

需要提取的字段:
{slots_text}

对话历史:
{history_text}

用户输入: {user_input}

返回JSON:
{{
  "slots": {{"字段名": "值或null"}},
  "query": "重组后的完整query，包含已知信息，标注缺失信息"
}}

只返回JSON。"""

    if llm is None:
        async with async_session_factory() as db:
            config = (await db.execute(select(ModelConfig).where(ModelConfig.is_default == True))).scalar_one_or_none()
            if config:
                llm = ChatOpenAI(model=config.model_name, api_key=config.api_key,
                                 base_url=config.api_endpoint, temperature=config.temperature)
            else:
                raise ValueError("No default model configured")

    response = await llm.ainvoke(prompt)
    return json.loads(response.content)


async def extract_and_rewrite(state: IntentState) -> dict:
    slot_defs = state.get("slot_definitions", [])

    if not slot_defs:
        return {
            "extracted_slots": {},
            "accumulated_slots": state.get("accumulated_slots", {}),
            "rewritten_query": state["user_input"],
        }

    result = await call_llm_for_extraction(
        user_input=state["user_input"],
        chat_history=state.get("chat_history", []),
        skill_name=state.get("l2_skill_name", ""),
        slot_definitions=slot_defs,
    )

    extracted = result.get("slots", {})
    # Merge into accumulated, skip None values
    accumulated = dict(state.get("accumulated_slots", {}))
    for k, v in extracted.items():
        if v is not None:
            accumulated[k] = v

    return {
        "extracted_slots": extracted,
        "accumulated_slots": accumulated,
        "rewritten_query": result.get("query", state["user_input"]),
    }
```

- [ ] **Step 4: Run test**

```bash
pytest tests/test_engine/test_extract_rewrite.py -v
# Expected: 1 passed
```

- [ ] **Step 5: Commit**

```bash
git add backend/engine/nodes/extract_rewrite.py tests/test_engine/test_extract_rewrite.py
git commit -m "feat(engine): add slot extraction and query rewrite node"
```

---

### Task 14: Intent switch detection node

**Files:**
- Create: `backend/engine/nodes/intent_switch.py`
- Test: `tests/test_engine/test_intent_switch.py`

- [ ] **Step 1: Write test**

```python
# tests/test_engine/test_intent_switch.py
from unittest.mock import patch

from backend.engine.nodes.intent_switch import intent_switch_check


async def test_continue_current_flow():
    state = {
        "user_input": "工资卡",
        "current_phase": "in_chatflow",
        "current_intent": "转账",
        "chat_history": [{"role": "assistant", "content": "请问从哪个账户转出？"}],
    }
    mock_result = {"continue": True, "reason": "用户在回答转账流程的问题"}

    with patch("backend.engine.nodes.intent_switch.call_llm_for_switch_check", return_value=mock_result):
        result = await intent_switch_check(state)

    assert result["intent_switched"] is False


async def test_detect_intent_switch():
    state = {
        "user_input": "算了不转了，帮我查下余额",
        "current_phase": "in_chatflow",
        "current_intent": "转账",
        "chat_history": [{"role": "assistant", "content": "请确认转账信息"}],
    }
    mock_result = {"continue": False, "reason": "用户取消转账，想查余额"}

    with patch("backend.engine.nodes.intent_switch.call_llm_for_switch_check", return_value=mock_result):
        result = await intent_switch_check(state)

    assert result["intent_switched"] is True
    assert result["current_phase"] == "idle"
```

- [ ] **Step 2: Run test to verify it fails**

```bash
pytest tests/test_engine/test_intent_switch.py -v
# Expected: FAIL
```

- [ ] **Step 3: Write intent_switch.py**

```python
# backend/engine/nodes/intent_switch.py
import json

from langchain_openai import ChatOpenAI
from sqlalchemy import select

from backend.core.database import async_session_factory
from backend.engine.state import IntentState
from backend.models.model_config import ModelConfig


async def call_llm_for_switch_check(user_input: str, current_intent: str,
                                     chat_history: list[dict], llm: ChatOpenAI | None = None) -> dict:
    history_text = "\n".join(f"{m['role']}: {m['content']}" for m in chat_history[-5:]) if chat_history else "(无)"

    prompt = f"""你是一个对话流转判断助手。当前用户正在"{current_intent}"流程中。

最近对话:
{history_text}

用户新输入: {user_input}

判断用户是在继续当前"{current_intent}"流程，还是想切换到其他话题？

返回JSON:
{{"continue": true/false, "reason": "简要理由"}}

只返回JSON。"""

    if llm is None:
        async with async_session_factory() as db:
            config = (await db.execute(select(ModelConfig).where(ModelConfig.is_default == True))).scalar_one_or_none()
            if config:
                llm = ChatOpenAI(model=config.model_name, api_key=config.api_key,
                                 base_url=config.api_endpoint, temperature=config.temperature)
            else:
                raise ValueError("No default model configured")

    response = await llm.ainvoke(prompt)
    return json.loads(response.content)


async def intent_switch_check(state: IntentState) -> dict:
    result = await call_llm_for_switch_check(
        user_input=state["user_input"],
        current_intent=state.get("current_intent", ""),
        chat_history=state.get("chat_history", []),
    )

    if result["continue"]:
        return {"intent_switched": False}

    # Intent switched — reset state for re-classification
    return {
        "intent_switched": True,
        "current_phase": "idle",
        "l1_category_id": None,
        "l1_category_name": None,
        "l2_chatflow_id": None,
        "l2_skill_name": None,
        "accumulated_slots": {},
    }
```

- [ ] **Step 4: Run test**

```bash
pytest tests/test_engine/test_intent_switch.py -v
# Expected: 2 passed
```

- [ ] **Step 5: Commit**

```bash
git add backend/engine/nodes/intent_switch.py tests/test_engine/test_intent_switch.py
git commit -m "feat(engine): add intent switch detection node"
```

---

### Task 15: ChatFlow call node + result processing

**Files:**
- Create: `backend/engine/nodes/call_chatflow.py`
- Create: `backend/engine/nodes/process_result.py`

- [ ] **Step 1: Write call_chatflow.py**

```python
# backend/engine/nodes/call_chatflow.py
import json

import httpx

from backend.engine.state import IntentState


async def call_chatflow(state: IntentState) -> dict:
    endpoint = state["chatflow_endpoint"]
    protocol = state.get("chatflow_protocol", "http_json")
    auth_type = state.get("chatflow_auth_type", "none")
    auth_credential = state.get("chatflow_auth_credential")

    headers = {"Content-Type": "application/json"}
    if auth_type == "bearer" and auth_credential:
        headers["Authorization"] = f"Bearer {auth_credential}"
    elif auth_type == "api_key" and auth_credential:
        headers["X-API-Key"] = auth_credential

    payload = {
        "query": state.get("rewritten_query", state["user_input"]),
        "conversation_id": state["conversation_id"],
        "slots": state.get("accumulated_slots", {}),
        "metadata": {"source": "intent_hub"},
    }

    try:
        if protocol == "sse":
            return await _call_sse(endpoint, headers, payload, state)
        else:
            return await _call_http_json(endpoint, headers, payload)
    except Exception as e:
        return {"chatflow_response": None, "error": f"ChatFlow call failed: {str(e)}"}


async def _call_http_json(endpoint: str, headers: dict, payload: dict) -> dict:
    async with httpx.AsyncClient(timeout=30.0) as client:
        resp = await client.post(endpoint, headers=headers, json=payload)
        resp.raise_for_status()
        data = resp.json()
        content = data.get("content", data.get("message", json.dumps(data)))
        return {"chatflow_response": content, "error": None}


async def _call_sse(endpoint: str, headers: dict, payload: dict, state: IntentState) -> dict:
    chunks = []
    sse_send = state.get("sse_send")

    async with httpx.AsyncClient(timeout=60.0) as client:
        async with client.stream("POST", endpoint, headers=headers, json=payload) as resp:
            resp.raise_for_status()
            async for line in resp.aiter_lines():
                if line.startswith("data:"):
                    data = line[5:].strip()
                    if data == "[DONE]":
                        break
                    chunks.append(data)
                    # Transparent proxy: forward SSE chunk to client
                    if sse_send:
                        await sse_send({"event": "message_delta", "data": json.dumps({"content": data})})

    full_response = "".join(chunks)
    return {"chatflow_response": full_response, "error": None}
```

- [ ] **Step 2: Write process_result.py**

```python
# backend/engine/nodes/process_result.py
from backend.engine.state import IntentState


async def process_result(state: IntentState) -> dict:
    response = state.get("chatflow_response", "")
    chat_history = list(state.get("chat_history", []))

    # Add user message to history
    chat_history.append({"role": "user", "content": state["user_input"], "source": "user"})

    # Add ChatFlow response to history
    if response:
        chat_history.append({
            "role": "assistant",
            "content": response,
            "source": "chatflow",
            "chatflow_id": state.get("l2_chatflow_id"),
        })

    return {
        "chat_history": chat_history,
        "current_phase": "in_chatflow",
        "error": None,
    }
```

- [ ] **Step 3: Commit**

```bash
git add backend/engine/nodes/call_chatflow.py backend/engine/nodes/process_result.py
git commit -m "feat(engine): add ChatFlow caller and result processing nodes"
```

---

### Task 16: Assemble full LangGraph and integration test

**Files:**
- Modify: `backend/engine/graph.py`
- Test: `tests/test_engine/test_graph.py`

- [ ] **Step 1: Write integration test**

```python
# tests/test_engine/test_graph.py
from unittest.mock import AsyncMock, patch

from backend.engine.graph import build_and_compile_graph
from backend.engine.state import IntentState


async def test_full_graph_new_conversation():
    """Test: user sends first message → L1 → expand → L2 → extract → call → result"""

    mock_categories = [
        {"id": 1, "name": "金融交易服务", "description": "转账缴费等"},
        {"id": 2, "name": "账户管理服务", "description": "查询余额等"},
    ]
    mock_skills = [
        {"id": 1, "chatflow_id": 101, "skill_name": "转账", "skill_desc": "用户发起转账",
         "extract_slots": [{"name": "payee", "type": "str", "desc": "收款人", "required": True}],
         "endpoint": "http://x", "protocol": "http_json", "auth_type": "none", "auth_credential": None},
    ]

    with patch("backend.engine.nodes.classify_l1.load_active_categories", return_value=mock_categories), \
         patch("backend.engine.nodes.classify_l1.call_llm_for_classification",
               return_value={"category_id": 1, "category_name": "金融交易服务", "confidence": 0.96}), \
         patch("backend.engine.nodes.expand_skills.load_skills_for_category", return_value=mock_skills), \
         patch("backend.engine.nodes.extract_rewrite.call_llm_for_extraction",
               return_value={"slots": {"payee": "张三"}, "query": "用户要转账给张三"}), \
         patch("backend.engine.nodes.call_chatflow.call_chatflow",
               return_value={"chatflow_response": "请问转多少钱？", "error": None}):

        graph = build_and_compile_graph()
        initial_state: IntentState = {
            "user_input": "我想转钱给张三",
            "conversation_id": "test-001",
            "chat_history": [],
            "current_phase": "idle",
            "accumulated_slots": {},
        }

        result = await graph.ainvoke(initial_state)

    assert result["l1_category_name"] == "金融交易服务"
    assert result["l2_skill_name"] == "转账"
    assert result["chatflow_response"] == "请问转多少钱？"
    assert result["current_phase"] == "in_chatflow"
    assert result["accumulated_slots"]["payee"] == "张三"
```

- [ ] **Step 2: Run test to verify it fails**

```bash
pytest tests/test_engine/test_graph.py -v
# Expected: FAIL - build_and_compile_graph not found
```

- [ ] **Step 3: Update graph.py with full assembly**

```python
# backend/engine/graph.py
from langgraph.graph import END, StateGraph

from backend.engine.nodes.call_chatflow import call_chatflow
from backend.engine.nodes.classify_l1 import classify_l1
from backend.engine.nodes.classify_l2 import classify_l2
from backend.engine.nodes.expand_skills import expand_skills
from backend.engine.nodes.extract_rewrite import extract_and_rewrite
from backend.engine.nodes.intent_switch import intent_switch_check
from backend.engine.nodes.process_result import process_result
from backend.engine.state import IntentState


def route_entry(state: IntentState) -> str:
    if state.get("current_phase") == "in_chatflow":
        return "intent_switch_check"
    return "classify_l1"


def route_after_switch_check(state: IntentState) -> str:
    if state.get("intent_switched"):
        return "classify_l1"
    return "call_chatflow"


def build_and_compile_graph():
    graph = StateGraph(IntentState)

    graph.add_node("intent_switch_check", intent_switch_check)
    graph.add_node("classify_l1", classify_l1)
    graph.add_node("expand_skills", expand_skills)
    graph.add_node("classify_l2", classify_l2)
    graph.add_node("extract_and_rewrite", extract_and_rewrite)
    graph.add_node("call_chatflow", call_chatflow)
    graph.add_node("process_result", process_result)

    graph.set_conditional_entry_point(route_entry, {
        "classify_l1": "classify_l1",
        "intent_switch_check": "intent_switch_check",
    })

    graph.add_conditional_edges("intent_switch_check", route_after_switch_check, {
        "classify_l1": "classify_l1",
        "call_chatflow": "call_chatflow",
    })

    graph.add_edge("classify_l1", "expand_skills")
    graph.add_edge("expand_skills", "classify_l2")
    graph.add_edge("classify_l2", "extract_and_rewrite")
    graph.add_edge("extract_and_rewrite", "call_chatflow")
    graph.add_edge("call_chatflow", "process_result")
    graph.add_edge("process_result", END)

    return graph.compile()
```

- [ ] **Step 4: Run integration test**

```bash
pytest tests/test_engine/test_graph.py -v
# Expected: 1 passed
```

- [ ] **Step 5: Run all tests**

```bash
pytest tests/ -v
# Expected: all passed
```

- [ ] **Step 6: Commit**

```bash
git add backend/engine/graph.py tests/test_engine/test_graph.py
git commit -m "feat(engine): assemble full LangGraph intent routing pipeline"
```

---

## Phase 5: External Conversation API

### Task 17: Conversation service

**Files:**
- Create: `backend/services/conversation_service.py`

- [ ] **Step 1: Write conversation_service.py**

```python
# backend/services/conversation_service.py
import uuid

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.models.conversation import Conversation
from backend.models.intent_trace import IntentTrace
from backend.models.message import Message


class ConversationService:
    def __init__(self, db: AsyncSession):
        self.db = db

    async def create(self, user_id: str, metadata: dict | None = None) -> Conversation:
        conv = Conversation(
            external_id=str(uuid.uuid4())[:16],
            user_id=user_id,
            status="active",
        )
        self.db.add(conv)
        await self.db.commit()
        await self.db.refresh(conv)
        return conv

    async def get_by_external_id(self, external_id: str) -> Conversation | None:
        result = await self.db.execute(
            select(Conversation).where(Conversation.external_id == external_id)
        )
        return result.scalar_one_or_none()

    async def update_state(self, conv_id: int, *, current_intent: str | None = None,
                           current_chatflow_id: int | None = None, slot_state: dict | None = None):
        conv = await self.db.get(Conversation, conv_id)
        if current_intent is not None:
            conv.current_intent = current_intent
        if current_chatflow_id is not None:
            conv.current_chatflow_id = current_chatflow_id
        if slot_state is not None:
            conv.slot_state = slot_state
        await self.db.commit()

    async def reset(self, conv_id: int) -> str:
        conv = await self.db.get(Conversation, conv_id)
        conv.current_intent = None
        conv.current_chatflow_id = None
        conv.slot_state = None
        await self.db.commit()
        return str(uuid.uuid4())[:8]  # section_id

    async def add_message(self, conv_id: int, role: str, content: str, source: str,
                          chatflow_id: int | None = None, intent_snapshot: dict | None = None) -> Message:
        msg = Message(
            conversation_id=conv_id, role=role, content=content, source=source,
            chatflow_id=chatflow_id, intent_snapshot=intent_snapshot,
        )
        self.db.add(msg)
        await self.db.commit()
        await self.db.refresh(msg)
        return msg

    async def get_messages(self, conv_id: int, limit: int = 50) -> list[Message]:
        result = await self.db.execute(
            select(Message).where(Message.conversation_id == conv_id)
            .order_by(Message.created_at.asc()).limit(limit)
        )
        return list(result.scalars().all())

    async def get_chat_history(self, conv_id: int, limit: int = 20) -> list[dict]:
        messages = await self.get_messages(conv_id, limit)
        return [{"role": m.role, "content": m.content, "source": m.source} for m in messages]

    async def add_intent_trace(self, conv_id: int, message_id: int, **kwargs) -> IntentTrace:
        trace = IntentTrace(conversation_id=conv_id, message_id=message_id, **kwargs)
        self.db.add(trace)
        await self.db.commit()
        return trace
```

- [ ] **Step 2: Commit**

```bash
git add backend/services/conversation_service.py
git commit -m "feat(service): add conversation lifecycle service"
```

---

### Task 18: SSE conversation chat endpoint

**Files:**
- Create: `backend/api/v1/conversation.py`
- Test: `tests/test_api/test_conversation_api.py`

- [ ] **Step 1: Write test**

```python
# tests/test_api/test_conversation_api.py
async def test_create_conversation(client):
    resp = await client.post("/api/v1/conversation/create", json={"user_id": "test-user"})
    assert resp.status_code == 200
    data = resp.json()
    assert "conversation_id" in data


async def test_get_messages_empty(client):
    resp = await client.post("/api/v1/conversation/create", json={"user_id": "test-user"})
    conv_id = resp.json()["conversation_id"]

    resp = await client.get(f"/api/v1/conversation/{conv_id}/messages")
    assert resp.status_code == 200
    assert resp.json()["messages"] == []


async def test_reset_conversation(client):
    resp = await client.post("/api/v1/conversation/create", json={"user_id": "test-user"})
    conv_id = resp.json()["conversation_id"]

    resp = await client.post(f"/api/v1/conversation/{conv_id}/reset")
    assert resp.status_code == 200
    assert "section_id" in resp.json()
```

- [ ] **Step 2: Run test to verify it fails**

```bash
pytest tests/test_api/test_conversation_api.py -v
# Expected: FAIL
```

- [ ] **Step 3: Write conversation.py**

```python
# backend/api/v1/conversation.py
import asyncio
import json

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from sse_starlette.sse import EventSourceResponse

from backend.api.deps import get_db_session
from backend.api.schemas.conversation import ChatRequest, ConversationCreate, ConversationOut, MessageOut
from backend.engine.graph import build_and_compile_graph
from backend.services.conversation_service import ConversationService

router = APIRouter()


@router.post("/conversation/create", response_model=ConversationOut)
async def create_conversation(body: ConversationCreate, db: AsyncSession = Depends(get_db_session)):
    svc = ConversationService(db)
    conv = await svc.create(user_id=body.user_id, metadata=body.metadata)
    return ConversationOut(conversation_id=conv.external_id, created_at=conv.created_at)


@router.post("/conversation/chat")
async def chat(body: ChatRequest, db: AsyncSession = Depends(get_db_session)):
    svc = ConversationService(db)
    conv = await svc.get_by_external_id(body.conversation_id)
    if not conv:
        return {"error": "Conversation not found"}

    chat_history = await svc.get_chat_history(conv.id)

    async def event_generator():
        queue = asyncio.Queue()

        async def sse_send(event: dict):
            await queue.put(event)

        initial_state = {
            "user_input": body.query,
            "conversation_id": body.conversation_id,
            "chat_history": chat_history,
            "current_phase": "in_chatflow" if conv.current_chatflow_id else "idle",
            "current_intent": conv.current_intent or "",
            "accumulated_slots": conv.slot_state or {},
            "sse_send": sse_send,
        }

        graph = build_and_compile_graph()

        async def run_graph():
            try:
                result = await graph.ainvoke(initial_state)
                # Save results to DB
                user_msg = await svc.add_message(conv.id, "user", body.query, "user")

                if result.get("chatflow_response"):
                    await svc.add_message(
                        conv.id, "assistant", result["chatflow_response"], "chatflow",
                        chatflow_id=result.get("l2_chatflow_id"),
                        intent_snapshot={
                            "l1": result.get("l1_category_name"),
                            "l2": result.get("l2_skill_name"),
                        },
                    )

                await svc.update_state(
                    conv.id,
                    current_intent=result.get("l2_skill_name"),
                    current_chatflow_id=result.get("l2_chatflow_id"),
                    slot_state=result.get("accumulated_slots"),
                )

                if result.get("l1_category_name"):
                    await svc.add_intent_trace(
                        conv.id, user_msg.id, level="l1",
                        input_text=body.query,
                        category_id=result.get("l1_category_id"),
                        confidence=result.get("l1_confidence"),
                    )

                if result.get("l2_chatflow_id"):
                    await svc.add_intent_trace(
                        conv.id, user_msg.id, level="l2",
                        input_text=body.query,
                        chatflow_id=result.get("l2_chatflow_id"),
                        confidence=result.get("l2_confidence"),
                        slots_extracted=result.get("extracted_slots"),
                        rewritten_query=result.get("rewritten_query"),
                    )

                # Send final events
                await queue.put({
                    "event": "intent",
                    "data": json.dumps({"l1": result.get("l1_category_name"), "l2": result.get("l2_skill_name")}),
                })
                if result.get("chatflow_response"):
                    await queue.put({
                        "event": "message_end",
                        "data": json.dumps({"content": result["chatflow_response"], "finish_reason": "stop"}),
                    })
            except Exception as e:
                await queue.put({"event": "error", "data": json.dumps({"error": str(e)})})
            finally:
                await queue.put(None)  # Signal done

        asyncio.create_task(run_graph())

        while True:
            event = await queue.get()
            if event is None:
                break
            yield event

    return EventSourceResponse(event_generator())


@router.get("/conversation/{conv_id}/messages")
async def get_messages(conv_id: str, db: AsyncSession = Depends(get_db_session)):
    svc = ConversationService(db)
    conv = await svc.get_by_external_id(conv_id)
    if not conv:
        return {"error": "Conversation not found"}
    messages = await svc.get_messages(conv.id)
    return {"messages": [MessageOut.model_validate(m) for m in messages]}


@router.post("/conversation/{conv_id}/reset")
async def reset_conversation(conv_id: str, db: AsyncSession = Depends(get_db_session)):
    svc = ConversationService(db)
    conv = await svc.get_by_external_id(conv_id)
    if not conv:
        return {"error": "Conversation not found"}
    section_id = await svc.reset(conv.id)
    return {"section_id": section_id}


@router.post("/conversation/{conv_id}/break")
async def break_conversation(conv_id: str):
    # TODO: implement cancellation via task tracking
    return {"success": True}
```

- [ ] **Step 4: Register router in main.py**

Add to `backend/main.py`:

```python
from backend.api.v1 import conversation as v1_conversation

app.include_router(v1_conversation.router, prefix="/api/v1", tags=["conversation"])
```

- [ ] **Step 5: Run tests**

```bash
pytest tests/test_api/test_conversation_api.py -v
# Expected: 3 passed
```

- [ ] **Step 6: Commit**

```bash
git add backend/api/v1/conversation.py backend/services/conversation_service.py tests/
git commit -m "feat(api): add SSE conversation chat endpoint with intent engine integration"
```

---

## Phase 6: Admin log, analytics, and Docker

### Task 19: Conversation log and analytics admin APIs

**Files:**
- Create: `backend/services/analytics_service.py`
- Create: `backend/api/admin/conversation_log.py`
- Create: `backend/api/admin/analytics.py`

- [ ] **Step 1: Write analytics_service.py**

```python
# backend/services/analytics_service.py
from datetime import datetime, timedelta

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.models.conversation import Conversation
from backend.models.intent_trace import IntentTrace
from backend.models.message import Message


class AnalyticsService:
    def __init__(self, db: AsyncSession):
        self.db = db

    async def overview(self) -> dict:
        today = datetime.utcnow().replace(hour=0, minute=0, second=0, microsecond=0)

        conv_count = (await self.db.execute(
            select(func.count()).select_from(Conversation).where(Conversation.created_at >= today)
        )).scalar() or 0

        msg_count = (await self.db.execute(
            select(func.count()).select_from(Message).where(Message.created_at >= today)
        )).scalar() or 0

        avg_latency = (await self.db.execute(
            select(func.avg(IntentTrace.latency_ms)).where(IntentTrace.created_at >= today)
        )).scalar()

        avg_confidence = (await self.db.execute(
            select(func.avg(IntentTrace.confidence)).where(IntentTrace.created_at >= today)
        )).scalar()

        return {
            "today_conversations": conv_count,
            "today_messages": msg_count,
            "avg_latency_ms": round(avg_latency or 0, 1),
            "avg_confidence": round(avg_confidence or 0, 3),
        }

    async def intent_distribution(self, days: int = 7) -> list[dict]:
        since = datetime.utcnow() - timedelta(days=days)
        result = await self.db.execute(
            select(IntentTrace.category_id, func.count().label("count"))
            .where(IntentTrace.level == "l1", IntentTrace.created_at >= since)
            .group_by(IntentTrace.category_id)
            .order_by(func.count().desc())
        )
        return [{"category_id": row.category_id, "count": row.count} for row in result.all()]
```

- [ ] **Step 2: Write conversation_log admin API**

```python
# backend/api/admin/conversation_log.py
from fastapi import APIRouter, Depends
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.api.deps import get_db_session
from backend.api.schemas.common import PageResponse
from backend.api.schemas.conversation import IntentTraceOut, MessageOut
from backend.models.conversation import Conversation
from backend.models.intent_trace import IntentTrace
from backend.models.message import Message

router = APIRouter()


@router.get("/conversations")
async def list_conversations(page: int = 1, page_size: int = 20, status: str | None = None,
                              db: AsyncSession = Depends(get_db_session)):
    query = select(Conversation)
    count_query = select(func.count()).select_from(Conversation)

    if status:
        query = query.where(Conversation.status == status)
        count_query = count_query.where(Conversation.status == status)

    total = (await db.execute(count_query)).scalar() or 0
    query = query.order_by(Conversation.created_at.desc()).offset((page - 1) * page_size).limit(page_size)
    result = await db.execute(query)
    items = result.scalars().all()

    return PageResponse(total=total, page=page, page_size=page_size, items=[
        {
            "id": c.id, "external_id": c.external_id, "user_id": c.user_id,
            "status": c.status, "current_intent": c.current_intent, "created_at": c.created_at,
        } for c in items
    ])


@router.get("/conversations/{conv_id}")
async def get_conversation_detail(conv_id: int, db: AsyncSession = Depends(get_db_session)):
    conv = await db.get(Conversation, conv_id)
    messages = (await db.execute(
        select(Message).where(Message.conversation_id == conv_id).order_by(Message.created_at)
    )).scalars().all()
    traces = (await db.execute(
        select(IntentTrace).where(IntentTrace.conversation_id == conv_id).order_by(IntentTrace.created_at)
    )).scalars().all()

    return {
        "conversation": {
            "id": conv.id, "external_id": conv.external_id, "user_id": conv.user_id,
            "status": conv.status, "current_intent": conv.current_intent, "slot_state": conv.slot_state,
        },
        "messages": [MessageOut.model_validate(m) for m in messages],
        "intent_traces": [IntentTraceOut.model_validate(t) for t in traces],
    }


@router.get("/conversations/{conv_id}/intent-trace", response_model=list[IntentTraceOut])
async def get_intent_trace(conv_id: int, db: AsyncSession = Depends(get_db_session)):
    result = await db.execute(
        select(IntentTrace).where(IntentTrace.conversation_id == conv_id).order_by(IntentTrace.created_at)
    )
    return result.scalars().all()
```

- [ ] **Step 3: Write analytics admin API**

```python
# backend/api/admin/analytics.py
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from backend.api.deps import get_db_session
from backend.services.analytics_service import AnalyticsService

router = APIRouter()


@router.get("/analytics/overview")
async def analytics_overview(db: AsyncSession = Depends(get_db_session)):
    svc = AnalyticsService(db)
    return await svc.overview()


@router.get("/analytics/intents")
async def analytics_intents(days: int = 7, db: AsyncSession = Depends(get_db_session)):
    svc = AnalyticsService(db)
    return await svc.intent_distribution(days=days)
```

- [ ] **Step 4: Register routers in main.py**

Add to `backend/main.py`:

```python
from backend.api.admin import conversation_log as admin_conv_log
from backend.api.admin import analytics as admin_analytics

app.include_router(admin_conv_log.router, prefix="/api/admin", tags=["admin-logs"])
app.include_router(admin_analytics.router, prefix="/api/admin", tags=["admin-analytics"])
```

- [ ] **Step 5: Run all tests**

```bash
pytest tests/ -v
# Expected: all passed
```

- [ ] **Step 6: Commit**

```bash
git add backend/services/analytics_service.py backend/api/admin/conversation_log.py backend/api/admin/analytics.py
git commit -m "feat(admin): add conversation log and analytics APIs"
```

---

### Task 20: Docker setup

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`

- [ ] **Step 1: Write Dockerfile**

```dockerfile
# Dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY pyproject.toml .
RUN pip install --no-cache-dir .

COPY backend/ backend/
COPY alembic/ alembic/
COPY alembic.ini .

EXPOSE 8000

CMD ["sh", "-c", "alembic upgrade head && uvicorn backend.main:app --host 0.0.0.0 --port 8000"]
```

- [ ] **Step 2: Write docker-compose.yml**

```yaml
# docker-compose.yml
version: "3.8"

services:
  intent-hub-api:
    build: .
    ports:
      - "8000:8000"
    environment:
      DATABASE_URL: mysql+aiomysql://root:intent_hub_pass@mysql:3306/intent_hub
      DATABASE_URL_SYNC: mysql+pymysql://root:intent_hub_pass@mysql:3306/intent_hub
      REDIS_URL: redis://redis:6379/0
      SECRET_KEY: ${SECRET_KEY:-change-me-in-production}
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_started

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: intent_hub_pass
      MYSQL_DATABASE: intent_hub
    ports:
      - "3307:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"

volumes:
  mysql_data:
```

- [ ] **Step 3: Test build**

```bash
cd ~/projects/ynet/intent-hub
docker compose build intent-hub-api
# Expected: build succeeds
```

- [ ] **Step 4: Commit**

```bash
git add Dockerfile docker-compose.yml
git commit -m "feat(deploy): add Dockerfile and docker-compose for local development"
```

---

### Task 21: Run full stack and smoke test

- [ ] **Step 1: Start services**

```bash
cd ~/projects/ynet/intent-hub
docker compose up -d
sleep 10
```

- [ ] **Step 2: Health check**

```bash
curl http://localhost:8000/api/v1/health
# Expected: {"status":"ok","service":"intent-hub"}
```

- [ ] **Step 3: Create a ChatFlow**

```bash
curl -X POST http://localhost:8000/api/admin/chatflows \
  -H "Content-Type: application/json" \
  -d '{"name":"转账服务","description":"处理用户的转账需求","endpoint":"http://localhost:9000/transfer","auth_type":"none","protocol":"http_json"}'
# Expected: 200 with chatflow object
```

- [ ] **Step 4: Create a category and skill**

```bash
curl -X POST http://localhost:8000/api/admin/categories \
  -H "Content-Type: application/json" \
  -d '{"name":"金融交易服务","description":"处理转账、缴费、兑换等金融交易"}'
# Note the category id from response

curl -X POST http://localhost:8000/api/admin/categories/{CATEGORY_ID}/skills \
  -H "Content-Type: application/json" \
  -d '{"chatflow_id":{CHATFLOW_ID},"skill_name":"转账","skill_desc":"用户发起转账请求","extract_slots":[{"name":"payee","type":"str","desc":"收款人","required":true}]}'
```

- [ ] **Step 5: Create a conversation**

```bash
curl -X POST http://localhost:8000/api/v1/conversation/create \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test-user"}'
# Expected: 200 with conversation_id
```

- [ ] **Step 6: Verify admin APIs**

```bash
curl http://localhost:8000/api/admin/chatflows
curl http://localhost:8000/api/admin/categories
curl http://localhost:8000/api/admin/analytics/overview
# Expected: all return 200
```

- [ ] **Step 7: Stop services**

```bash
docker compose down
```

- [ ] **Step 8: Final commit**

```bash
git add -A
git commit -m "chore: smoke test passed, intent-hub backend v0.1.0 ready"
```

---

## Summary

| Phase | Tasks | What it delivers |
|-------|-------|-----------------|
| 1. Scaffolding | 1-3 | Python project, FastAPI app, config, health check |
| 2. Models | 4-5 | 8 ORM models, Alembic migrations |
| 3. Admin CRUD | 6-9 | ChatFlow/Category/Model/Prompt management APIs |
| 4. Engine | 10-16 | Full LangGraph intent routing pipeline |
| 5. Conversation | 17-18 | SSE chat endpoint, conversation lifecycle |
| 6. Deploy | 19-21 | Log/analytics APIs, Docker, smoke test |

**Total: 21 tasks, ~100 steps**

**Next:** Frontend management UI plan (separate document).
