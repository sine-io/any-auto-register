import ast
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]

HEAVY_PLATFORMS = ("kiro", "grok", "chatgpt")
LIGHT_PLATFORMS = ("cursor", "trae")

PLATFORM_SPECS = {
    "cursor": {
        "class_name": "CursorPlatform",
        "helper_modules": {
            "_registration_service": "registration",
            "_account_service": "account",
            "_desktop_service": "desktop",
        },
    },
    "trae": {
        "class_name": "TraePlatform",
        "helper_modules": {
            "_registration_service": "registration",
            "_account_service": "account",
            "_desktop_service": "desktop",
            "_billing_service": "billing",
        },
    },
    "kiro": {
        "class_name": "KiroPlatform",
        "helper_modules": {
            "_registration_service": "registration",
            "_token_service": "token",
            "_desktop_service": "desktop",
            "_manager_sync_service": "manager_sync",
        },
    },
    "grok": {
        "class_name": "GrokPlatform",
        "helper_modules": {
            "_registration_service": "registration",
            "_cookie_service": "cookie",
            "_sync_service": "sync",
        },
    },
    "chatgpt": {
        "class_name": "ChatGPTPlatform",
        "helper_modules": {
            "_registration_service": "registration",
            "_token_service": "token",
            "_billing_service": "billing",
            "_external_sync_service": "external_sync",
        },
    },
}


def _module_tree(path: Path) -> ast.Module:
    return ast.parse(path.read_text(encoding="utf-8"), filename=str(path))


def _plugin_path(platform: str) -> Path:
    return REPO_ROOT / "platforms" / platform / "plugin.py"


def _services_init_path(platform: str) -> Path:
    return REPO_ROOT / "platforms" / platform / "services" / "__init__.py"


def _class_node(tree: ast.Module, class_name: str) -> ast.ClassDef:
    for node in tree.body:
        if isinstance(node, ast.ClassDef) and node.name == class_name:
            return node
    raise AssertionError(f"class {class_name} not found")


def _class_methods(class_node: ast.ClassDef) -> dict[str, ast.FunctionDef]:
    return {
        node.name: node
        for node in class_node.body
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
    }


def _top_level_import_modules(tree: ast.Module) -> set[str]:
    modules = set()
    for node in tree.body:
        if isinstance(node, ast.Import):
            modules.update(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module is not None:
            modules.add(node.module)
    return modules


def _local_import_modules(func_node: ast.FunctionDef) -> set[str]:
    modules = set()
    for node in ast.walk(func_node):
        if node is func_node:
            continue
        if isinstance(node, ast.Import):
            modules.update(alias.name for alias in node.names)
        elif isinstance(node, ast.ImportFrom) and node.module is not None:
            modules.add(node.module)
    return modules


def _top_level_relative_import_modules(tree: ast.Module) -> set[str]:
    modules = set()
    for node in tree.body:
        if isinstance(node, ast.ImportFrom) and node.level and node.module is not None:
            modules.add(node.module)
    return modules


def _has_top_level_assignment(tree: ast.Module, name: str) -> bool:
    for node in tree.body:
        if not isinstance(node, ast.Assign):
            continue
        for target in node.targets:
            if isinstance(target, ast.Name) and target.id == name:
                return True
    return False


def _has_top_level_function(tree: ast.Module, name: str) -> bool:
    return any(isinstance(node, ast.FunctionDef) and node.name == name for node in tree.body)


def test_platform_plugins_expose_service_factory_helpers():
    for platform, spec in PLATFORM_SPECS.items():
        tree = _module_tree(_plugin_path(platform))
        methods = _class_methods(_class_node(tree, spec["class_name"]))

        for helper_name in spec["helper_modules"]:
            assert helper_name in methods, f"{platform} plugin should define {helper_name}()"
            assert helper_name.endswith("_service"), f"{platform} helper {helper_name} should use the _service suffix"


def test_heavy_platform_plugins_use_local_service_imports():
    for platform in HEAVY_PLATFORMS:
        spec = PLATFORM_SPECS[platform]
        tree = _module_tree(_plugin_path(platform))
        methods = _class_methods(_class_node(tree, spec["class_name"]))
        service_prefix = f"platforms.{platform}.services"

        top_level_service_imports = {
            module for module in _top_level_import_modules(tree) if module.startswith(service_prefix)
        }
        assert not top_level_service_imports, (
            f"{platform} plugin should avoid top-level service imports; found {sorted(top_level_service_imports)}"
        )

        for helper_name, module_name in spec["helper_modules"].items():
            local_service_imports = {
                module for module in _local_import_modules(methods[helper_name]) if module.startswith(service_prefix)
            }
            expected_import = f"{service_prefix}.{module_name}"
            assert expected_import in local_service_imports, (
                f"{platform} helper {helper_name} should locally import {expected_import}"
            )

    for platform in LIGHT_PLATFORMS:
        spec = PLATFORM_SPECS[platform]
        tree = _module_tree(_plugin_path(platform))
        methods = _class_methods(_class_node(tree, spec["class_name"]))
        service_prefix = f"platforms.{platform}.services"

        top_level_service_imports = {
            module for module in _top_level_import_modules(tree) if module.startswith(service_prefix)
        }
        assert top_level_service_imports == {service_prefix}, (
            f"{platform} plugin should keep the simple top-level service import strategy"
        )

        for helper_name in spec["helper_modules"]:
            local_service_imports = {
                module for module in _local_import_modules(methods[helper_name]) if module.startswith(service_prefix)
            }
            assert not local_service_imports, (
                f"{platform} helper {helper_name} should not need local service imports in the light strategy"
            )


def test_heavy_service_packages_use_lazy_exports():
    for platform in HEAVY_PLATFORMS:
        tree = _module_tree(_services_init_path(platform))

        assert _has_top_level_assignment(tree, "__all__"), f"{platform} services package should define __all__"
        assert _has_top_level_assignment(tree, "_SERVICE_MODULES"), (
            f"{platform} services package should map exports lazily via _SERVICE_MODULES"
        )
        assert _has_top_level_function(tree, "__getattr__"), (
            f"{platform} services package should expose lazy exports via __getattr__"
        )
        assert not _top_level_relative_import_modules(tree), (
            f"{platform} services package should avoid eager top-level relative imports"
        )

    for platform in LIGHT_PLATFORMS:
        tree = _module_tree(_services_init_path(platform))

        assert _has_top_level_assignment(tree, "__all__"), f"{platform} services package should define __all__"
        assert _top_level_relative_import_modules(tree), (
            f"{platform} services package should keep direct top-level relative exports"
        )
        assert not _has_top_level_assignment(tree, "_SERVICE_MODULES"), (
            f"{platform} services package should not require the heavy-platform lazy export map"
        )
        assert not _has_top_level_function(tree, "__getattr__"), (
            f"{platform} services package should not require module-level lazy exports"
        )
