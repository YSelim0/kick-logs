from fastapi.testclient import TestClient

from kick_logs.presentation.http.app import create_app


def test_health_route_returns_ok() -> None:
    client = TestClient(create_app())

    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok"}
