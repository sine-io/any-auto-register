def test_solver_status_and_restart_flow(api_get_json, api_post_json, wait_for_solver_status):
    status = api_get_json("/solver/status", timeout=10)
    assert "running" in status
    assert "status" in status
    assert "reason" in status
    assert status["status"] in {"starting", "running", "failed", "stopped"}

    initial = wait_for_solver_status()

    api_post_json("/solver/restart", {}, timeout=60)

    final = wait_for_solver_status()
    assert final["status"] in {"running", "failed", "stopped"}
    if final["status"] == "failed":
        assert final["reason"]
