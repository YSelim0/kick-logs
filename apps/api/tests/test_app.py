from fastapi import FastAPI
from httpx import ASGITransport, AsyncClient

from kick_logs.core.config import Settings
from kick_logs.presentation.http.app import create_app, parse_cors_origins


def test_create_app_returns_fastapi_instance() -> None:
    assert isinstance(create_app(seed_super_admin_on_startup=False), FastAPI)


def test_parse_cors_origins_trims_comma_separated_values() -> None:
    assert parse_cors_origins("http://localhost:3000, http://127.0.0.1:3000") == [
        "http://localhost:3000",
        "http://127.0.0.1:3000",
    ]


async def test_cors_preflight_allows_configured_frontend_origin() -> None:
    app = create_app(
        settings=Settings(backend_cors_origins="http://localhost:3000"),
        seed_super_admin_on_startup=False,
    )

    async with AsyncClient(
        transport=ASGITransport(app=app),
        base_url="http://testserver",
    ) as client:
        response = await client.options(
            "/auth/login",
            headers={
                "Origin": "http://localhost:3000",
                "Access-Control-Request-Method": "POST",
                "Access-Control-Request-Headers": "content-type",
            },
        )

    assert response.status_code == 200
    assert response.headers["access-control-allow-origin"] == "http://localhost:3000"
    assert response.headers["access-control-allow-credentials"] == "true"
