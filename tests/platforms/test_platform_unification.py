import ast
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]

HEAVY_PLATFORMS = ("kiro", "grok", "chatgpt")
LIGHT_PLATFORMS = ("cursor", "trae")
LAZY_EXPORT_PLATFORMS = ("kiro", "grok", "chatgpt")
DIRECT_EXPORT_PLATFORMS = ("cursor", "trae")

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


def _class_method_names_in_order(class_node: ast.ClassDef) -> list[str]:
    return [
        node.name
        for node in class_node.body
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))
    ]


def _top_level_import_nodes(tree: ast.Module) -> list[ast.stmt]:
    return [node for node in tree.body if isinstance(node, (ast.Import, ast.ImportFrom))]


def _local_import_nodes(func_node: ast.FunctionDef) -> list[ast.stmt]:
    return [
        node
        for node in ast.walk(func_node)
        if node is not func_node and isinstance(node, (ast.Import, ast.ImportFrom))
    ]


def _is_platform_service_import(node: ast.stmt, platform: str) -> bool:
    service_prefix = f"platforms.{platform}.services"
    platform_prefix = f"platforms.{platform}"

    if isinstance(node, ast.Import):
        return any(alias.name.startswith(service_prefix) for alias in node.names)

    if not isinstance(node, ast.ImportFrom):
        return False

    if node.level == 0:
        if node.module is None:
            return False
        if node.module.startswith(service_prefix):
            return True
        return node.module == platform_prefix and any(alias.name == "services" for alias in node.names)

    if node.level == 1:
        if node.module is None:
            return any(alias.name == "services" for alias in node.names)
        return node.module == "services" or node.module.startswith("services.")

    return False


def _service_import_statements(nodes: list[ast.stmt], platform: str) -> list[str]:
    return [ast.unparse(node) for node in nodes if _is_platform_service_import(node, platform)]


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


def test_platform_plugins_follow_recommended_method_order():
    for platform, spec in PLATFORM_SPECS.items():
        tree = _module_tree(_plugin_path(platform))
        method_names = _class_method_names_in_order(_class_node(tree, spec["class_name"]))
        method_positions = {name: index for index, name in enumerate(method_names)}
        helper_names = list(spec["helper_modules"])
        helper_positions = [method_positions[name] for name in helper_names]

        assert method_positions["__init__"] < min(helper_positions), (
            f"{platform} plugin should place __init__ before helper factories"
        )
        assert helper_positions == sorted(helper_positions), (
            f"{platform} plugin should keep helper factories in the documented order"
        )
        assert max(helper_positions) < method_positions["register"], (
            f"{platform} plugin should place helper factories before register"
        )
        assert method_positions["register"] < method_positions["check_valid"], (
            f"{platform} plugin should place register before check_valid"
        )
        assert method_positions["check_valid"] < method_positions["get_platform_actions"], (
            f"{platform} plugin should place check_valid before get_platform_actions"
        )
        assert method_positions["get_platform_actions"] < method_positions["execute_action"], (
            f"{platform} plugin should place get_platform_actions before execute_action"
        )


def test_platform_plugins_follow_service_import_discipline():
    for platform in HEAVY_PLATFORMS:
        spec = PLATFORM_SPECS[platform]
        tree = _module_tree(_plugin_path(platform))
        methods = _class_methods(_class_node(tree, spec["class_name"]))
        top_level_service_imports = _service_import_statements(_top_level_import_nodes(tree), platform)
        missing_local_imports = [
            helper_name
            for helper_name in spec["helper_modules"]
            if not _service_import_statements(_local_import_nodes(methods[helper_name]), platform)
        ]

        issues = []
        if top_level_service_imports:
            issues.append(f"avoid top-level service imports: {top_level_service_imports}")
        if missing_local_imports:
            issues.append(f"import services locally inside helpers: {missing_local_imports}")

        assert not issues, (
            f"{platform} heavy plugin should resolve services lazily; {'; '.join(issues)}"
        )

    for platform in LIGHT_PLATFORMS:
        spec = PLATFORM_SPECS[platform]
        tree = _module_tree(_plugin_path(platform))
        methods = _class_methods(_class_node(tree, spec["class_name"]))
        top_level_service_imports = _service_import_statements(_top_level_import_nodes(tree), platform)
        if top_level_service_imports:
            continue

        missing_local_imports = [
            helper_name
            for helper_name in spec["helper_modules"]
            if not _service_import_statements(_local_import_nodes(methods[helper_name]), platform)
        ]
        assert not missing_local_imports, (
            f"{platform} plugin should either keep eager top-level service imports "
            f"or import services locally in helpers; missing local imports in {missing_local_imports}"
        )


def test_heavy_service_packages_use_lazy_exports():
    for platform in LAZY_EXPORT_PLATFORMS:
        tree = _module_tree(_services_init_path(platform))
        contract_gaps = []
        if not _has_top_level_assignment(tree, "__all__"):
            contract_gaps.append("define __all__")
        if not _has_top_level_function(tree, "__getattr__"):
            contract_gaps.append("define __getattr__")
        eager_relative_imports = sorted(_top_level_relative_import_modules(tree))
        if eager_relative_imports:
            contract_gaps.append(f"avoid eager top-level relative imports: {eager_relative_imports}")

        assert not contract_gaps, (
            f"{platform} services package should honor the lazy-export contract; "
            f"{'; '.join(contract_gaps)}"
        )

    for platform in DIRECT_EXPORT_PLATFORMS:
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
