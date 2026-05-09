from fastapi import FastAPI

from kick_logs.presentation.http.app import create_app


def test_create_app_returns_fastapi_instance() -> None:
    assert isinstance(create_app(), FastAPI)
