from __future__ import annotations

import base64
import hashlib
import hmac
import json
import time
from typing import Any
from urllib.parse import parse_qs

from fastapi import APIRouter, Request, status
from fastapi.responses import JSONResponse
from pydantic import BaseModel
from sqlalchemy import text
from sqlalchemy.exc import IntegrityError
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from app.config import AuthConfig


class RegisterRequest(BaseModel):
    username: str = ""
    password: str = ""
    role: str = "user"


def _b64url_encode(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode("ascii")


def _b64url_decode(data: str) -> bytes:
    padding = "=" * (-len(data) % 4)
    return base64.urlsafe_b64decode(data + padding)


def _sign_hs256(unsigned: str, secret: str) -> str:
    sig = hmac.new(secret.encode("utf-8"), unsigned.encode("utf-8"), hashlib.sha256).digest()
    return _b64url_encode(sig)


def generate_jwt(secret: str, subject: str, role: str, scope: str, ttl_seconds: int) -> str:
    header_b64 = _b64url_encode(b'{"alg":"HS256","typ":"JWT"}')
    now = int(time.time())
    claims = {
        "sub": subject,
        "role": role,
        "scope": scope,
        "iss": "inventory-order-service",
        "iat": now,
        "exp": now + ttl_seconds,
    }
    claims_b64 = _b64url_encode(json.dumps(claims, separators=(",", ":")).encode("utf-8"))
    unsigned = f"{header_b64}.{claims_b64}"
    sig = _sign_hs256(unsigned, secret)
    return f"{unsigned}.{sig}"


def verify_jwt(token_str: str, secret: str) -> dict[str, Any]:
    parts = token_str.split(".")
    if len(parts) != 3:
        raise ValueError("invalid or expired JWT token")

    unsigned = f"{parts[0]}.{parts[1]}"
    expected_sig = _sign_hs256(unsigned, secret)
    if not hmac.compare_digest(parts[2], expected_sig):
        raise ValueError("invalid or expired JWT token")

    try:
        claims = json.loads(_b64url_decode(parts[1]).decode("utf-8"))
    except Exception as exc:
        raise ValueError("invalid or expired JWT token") from exc

    if int(time.time()) > int(claims.get("exp", 0)):
        raise ValueError("invalid or expired JWT token")

    return claims


class AuthHandler:
    """Manages OAuth 2.0 user registration (/oauth/register) and token issuance (/oauth/token) backed by PostgreSQL + pgcrypto Bcrypt."""

    def __init__(
        self,
        cfg: AuthConfig,
        session_factory: async_sessionmaker[AsyncSession],
    ) -> None:
        self.cfg = cfg
        self._session_factory = session_factory
        self.router = APIRouter(tags=["auth"])
        self.router.add_api_route("/oauth/config", self.get_config, methods=["GET"])
        self.router.add_api_route("/oauth/register", self.register, methods=["POST"])
        self.router.add_api_route("/oauth/token", self.issue_token, methods=["POST"])

    async def get_config(self) -> JSONResponse:
        return JSONResponse(
            status_code=status.HTTP_200_OK,
            content={"enable_register": self.cfg.enable_register},
        )

    async def register(self, req: RegisterRequest) -> JSONResponse:
        if not self.cfg.enable_register:
            return JSONResponse(
                status_code=status.HTTP_403_FORBIDDEN,
                content={"error": "user registration is disabled"},
            )
        username = req.username.strip()
        if not username or len(req.password) < 6:
            return JSONResponse(
                status_code=status.HTTP_400_BAD_REQUEST,
                content={"error": "username and password (min 6 chars) are required"},
            )
        role = req.role.strip() or "user"

        stmt = text(
            """
            INSERT INTO users (username, password_hash, role)
            VALUES (:username, crypt(:password, gen_salt('bf', 10)), :role)
            RETURNING id, username, role, created_at, updated_at
            """
        )
        async with self._session_factory() as session:
            try:
                result = await session.execute(
                    stmt,
                    {"username": username, "password": req.password, "role": role},
                )
                row = result.mappings().one()
                await session.commit()
                return JSONResponse(
                    status_code=status.HTTP_201_CREATED,
                    content={
                        "id": str(row["id"]),
                        "username": row["username"],
                        "role": row["role"],
                        "created_at": row["created_at"].isoformat(),
                        "updated_at": row["updated_at"].isoformat(),
                    },
                )
            except IntegrityError:
                await session.rollback()
                return JSONResponse(
                    status_code=status.HTTP_409_CONFLICT,
                    content={"error": "username already exists"},
                )

    async def issue_token(self, request: Request) -> JSONResponse:
        content_type = request.headers.get("content-type", "")
        raw_body = await request.body()

        payload: dict[str, str] = {}
        if "application/json" in content_type:
            try:
                data = json.loads(raw_body.decode("utf-8"))
                if isinstance(data, dict):
                    payload = {k: str(v) for k, v in data.items() if v is not None}
            except Exception:
                return JSONResponse(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    content={
                        "error": "invalid_request",
                        "error_description": "cannot parse OAuth 2.0 token request",
                    },
                )
        else:
            try:
                data = json.loads(raw_body.decode("utf-8"))
                if isinstance(data, dict):
                    payload = {k: str(v) for k, v in data.items() if v is not None}
            except Exception:
                parsed = parse_qs(raw_body.decode("utf-8"))
                payload = {k: v[0] for k, v in parsed.items() if v}

        # Support HTTP Basic Auth header for client_id / client_secret
        auth_header = request.headers.get("authorization", "")
        if auth_header.startswith("Basic "):
            try:
                decoded = base64.b64decode(auth_header[6:]).decode("utf-8")
                if ":" in decoded:
                    cid, csec = decoded.split(":", 1)
                    payload["client_id"] = cid
                    payload["client_secret"] = csec
            except Exception:
                pass

        grant_type = payload.get("grant_type", "")
        scope = payload.get("scope") or "inventory:read inventory:write orders:read orders:write"

        if grant_type == "password":
            stmt = text(
                """
                SELECT username, role
                FROM users
                WHERE username = :username AND password_hash = crypt(:password, password_hash)
                """
            )
            async with self._session_factory() as session:
                result = await session.execute(
                    stmt,
                    {
                        "username": payload.get("username", ""),
                        "password": payload.get("password", ""),
                    },
                )
                row = result.mappings().one_or_none()

            if row is None:
                return JSONResponse(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    content={
                        "error": "invalid_grant",
                        "error_description": "invalid username or password",
                    },
                )
            subject = str(row["username"])
            role = str(row["role"])

        elif grant_type == "client_credentials":
            stmt = text(
                """
                SELECT client_id, scope
                FROM oauth_clients
                WHERE client_id = :client_id AND client_secret_hash = crypt(:client_secret, client_secret_hash)
                """
            )
            async with self._session_factory() as session:
                result = await session.execute(
                    stmt,
                    {
                        "client_id": payload.get("client_id", ""),
                        "client_secret": payload.get("client_secret", ""),
                    },
                )
                row = result.mappings().one_or_none()

            if row is None:
                return JSONResponse(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    content={
                        "error": "invalid_client",
                        "error_description": "invalid client_id or client_secret",
                    },
                )
            subject = str(row["client_id"])
            role = "service"
            if not payload.get("scope") and row.get("scope"):
                scope = str(row["scope"])
        else:
            return JSONResponse(
                status_code=status.HTTP_400_BAD_REQUEST,
                content={
                    "error": "unsupported_grant_type",
                    "error_description": "supported grant_types are 'password' and 'client_credentials'",
                },
            )

        ttl = self.cfg.token_ttl
        token = generate_jwt(self.cfg.jwt_secret, subject, role, scope, ttl)
        return JSONResponse(
            status_code=status.HTTP_200_OK,
            content={
                "access_token": token,
                "token_type": "Bearer",
                "expires_in": ttl,
                "scope": scope,
            },
        )
