from __future__ import annotations

import pytest

from app.auth import generate_jwt, verify_jwt


def test_generate_and_verify_jwt_success() -> None:
    secret = "super-secret-jwt-key-for-unit-test"
    token = generate_jwt(secret, "admin", "admin", "inventory:read inventory:write", 600)

    claims = verify_jwt(token, secret)
    assert claims["sub"] == "admin"
    assert claims["role"] == "admin"
    assert claims["scope"] == "inventory:read inventory:write"


def test_verify_jwt_rejects_wrong_secret() -> None:
    secret = "super-secret-jwt-key-for-unit-test"
    token = generate_jwt(secret, "admin", "admin", "inventory:read", 600)

    with pytest.raises(ValueError, match="invalid or expired JWT token"):
        verify_jwt(token, "wrong-secret")


def test_verify_jwt_rejects_expired_token() -> None:
    secret = "super-secret-jwt-key-for-unit-test"
    token = generate_jwt(secret, "admin", "admin", "inventory:read", -10)

    with pytest.raises(ValueError, match="invalid or expired JWT token"):
        verify_jwt(token, secret)
