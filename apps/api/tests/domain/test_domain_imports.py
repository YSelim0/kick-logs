import ast
from pathlib import Path

FORBIDDEN_TOP_LEVEL_IMPORTS = {
    "asyncpg",
    "fastapi",
    "httpx",
    "pydantic",
    "sqlalchemy",
    "websockets",
}


def test_domain_layer_has_no_framework_imports() -> None:
    domain_root = Path("src/kick_logs/domain")

    for source_file in domain_root.rglob("*.py"):
        tree = ast.parse(source_file.read_text(encoding="utf-8"))
        imports = [
            node
            for node in ast.walk(tree)
            if isinstance(node, ast.Import | ast.ImportFrom)
        ]

        imported_roots: set[str] = set()
        for import_node in imports:
            if isinstance(import_node, ast.Import):
                imported_roots.update(alias.name.split(".", 1)[0] for alias in import_node.names)
            elif import_node.module:
                imported_roots.add(import_node.module.split(".", 1)[0])

        assert imported_roots.isdisjoint(FORBIDDEN_TOP_LEVEL_IMPORTS), source_file
