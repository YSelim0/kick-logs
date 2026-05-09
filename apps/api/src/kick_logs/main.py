import uvicorn

from kick_logs.core.config import get_settings
from kick_logs.presentation.http.app import create_app

app = create_app()


def main() -> None:
    settings = get_settings()
    uvicorn.run(
        "kick_logs.main:app",
        host=settings.api_host,
        port=settings.api_port,
        reload=settings.app_env == "local",
    )


if __name__ == "__main__":
    main()
