def test_control_plane_task_flow(api_get_json, api_post_json, api_get_text, wait_for_task):
    health = api_get_json("/health")
    assert health["status"] == "ok"

    platforms = api_get_json("/platforms")
    assert isinstance(platforms, list)
    assert platforms

    config = api_get_json("/config")
    assert isinstance(config, dict)

    created = api_post_json("/tasks/register", {"platform": "dummy", "count": 1})
    assert "task_id" in created

    task = wait_for_task(created["task_id"])
    assert task["status"] in {"done", "failed"}

    stream = api_get_text(f"/tasks/{created['task_id']}/logs/stream")
    assert "data:" in stream
