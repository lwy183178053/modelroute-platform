import json
import os
import shlex
import subprocess
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def read_script(name):
    return (ROOT / "deploy" / "scripts" / name).read_text(encoding="utf-8")


def shell_path(path):
    if os.name != "nt":
        return str(path)
    drive, tail = os.path.splitdrive(str(Path(path).resolve()))
    tail = tail.lstrip("\\").replace("\\", "/")
    return f"/mnt/{drive[0].lower()}/{tail}"


class ReleaseToolContractTest(unittest.TestCase):
    def test_operations_verifier_is_read_only_and_covers_phase_eight(self):
        verifier = read_script("verify-operations.sh")
        for path in (
            "/api/usage",
            "/api/admin/overview",
            "/api/admin/usage",
            "/api/admin/platform-settings",
            "/api/admin/groups",
            "/api/admin/redemptions",
            "/api/admin/system-tasks",
            "/api/admin/channels",
        ):
            self.assertIn(path, verifier)
        for write_path in (
            "/api/redemptions/redeem",
            "/api/admin/system-tasks/log-cleanup",
            "/api/admin/channels/$CHANNEL_ID/test",
            "/api/admin/channels/$CHANNEL_ID/balance/refresh",
        ):
            self.assertNotIn(write_path, verifier)

    def test_restore_drill_targets_generated_temporary_databases(self):
        script = read_script("verify-backup-restore.sh")
        self.assertIn("CONFIRM_RESTORE_DRILL=MODELROUTE_RESTORE_DRILL", script)
        self.assertIn("modelroute_restore_check_", script)
        self.assertIn("newapi_restore_check_", script)
        self.assertIn("pg_restore --exit-on-error", script)
        self.assertIn("FROM platform_schema_migrations", script)
        self.assertNotIn("FROM schema_migrations", script)
        self.assertIn('test "$PLATFORM_TEST_DB" != "$PLATFORM_DB_NAME"', script)
        self.assertIn('test "$NEWAPI_TEST_DB" != "$NEWAPI_DB_NAME"', script)
        self.assertNotIn('-d "$PLATFORM_DB_NAME" <', script)
        self.assertNotIn('-d "$NEWAPI_DB_NAME" <', script)

    def test_rollback_requires_confirmation_and_rejects_traversal(self):
        script = ROOT / "deploy" / "scripts" / "rollback-release.sh"
        result = subprocess.run(
            ["bash", shell_path(script), "../outside"],
            cwd=ROOT,
            capture_output=True,
        )
        self.assertNotEqual(result.returncode, 0)
        stderr = result.stderr.decode("utf-8", errors="ignore").replace("\x00", "")
        self.assertIn("CONFIRM_ROLLBACK=MODELROUTE_ROLLBACK", stderr)

        result = subprocess.run(
            [
                "bash",
                "-lc",
                "CONFIRM_ROLLBACK=MODELROUTE_ROLLBACK "
                + shlex.quote(shell_path(script))
                + " ../outside",
            ],
            cwd=ROOT,
            capture_output=True,
        )
        self.assertNotEqual(result.returncode, 0)
        stderr = result.stderr.decode("utf-8", errors="ignore").replace("\x00", "")
        self.assertIn("发布目录名", stderr)

    def test_load_test_has_bounded_read_only_defaults(self):
        script = read_script("load-test.sh")
        self.assertIn("/api/health/ready", script)
        self.assertIn("REQUESTS=${REQUESTS:-200}", script)
        self.assertIn("CONCURRENCY=${CONCURRENCY:-10}", script)
        self.assertIn("MAX_REQUESTS=10000", script)
        self.assertNotIn("Authorization:", script)
        self.assertNotIn("-X POST", script)

    def test_candidate_compose_is_isolated_and_default_image_is_pinned(self):
        result = subprocess.run(
            [
                "docker",
                "compose",
                "-f",
                "deploy/newapi-contract/compose.yaml",
                "config",
                "--format",
                "json",
            ],
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True,
        )
        config = json.loads(result.stdout)
        new_api = config["services"]["new-api"]
        self.assertIn("v1.0.0-rc.21@sha256:428018a3", new_api["image"])
        self.assertEqual(new_api["ports"][0]["host_ip"], "127.0.0.1")
        self.assertTrue(
            any(
                not config["networks"][name].get("internal", False)
                for name in new_api["networks"]
            )
        )
        self.assertEqual(list(config["services"]["postgres"]["networks"]), ["contract"])
        self.assertEqual(list(config["services"]["redis"]["networks"]), ["contract"])

    def test_candidate_contract_covers_required_protocols_and_accounting(self):
        verifier = read_script("verify-newapi-contract.sh")
        fixture = (ROOT / "deploy" / "newapi-contract" / "run_contract.py").read_text(
            encoding="utf-8"
        )
        combined = verifier + fixture
        self.assertIn('"New-Api-User"', fixture)
        self.assertIn('"/api/channel/fix"', fixture)
        self.assertIn("sys.argv[2]", fixture)
        self.assertIn("sys.argv[3]", fixture)
        self.assertIn("time.sleep(0.25)", fixture)
        for marker in (
            "/api/setup",
            "/api/user/login",
            "/api/user/",
            "/api/token/",
            "/api/channel/",
            "billing_setting.billing_mode",
            "billing_setting.billing_expr",
            "tiered_expr",
            "expr_b64",
            "matched_tier",
            "/v1/chat/completions",
            "/v1/responses",
            "/v1/embeddings",
            "stream",
            "cached_tokens",
            "refund",
            "no_available_channel",
        ):
            self.assertIn(marker, combined)

    def test_candidate_verifier_bridges_wsl_to_windows_loopback(self):
        verifier = read_script("verify-newapi-contract.sh")
        self.assertIn("python.exe", verifier)
        self.assertIn("wslpath -w", verifier)
        self.assertIn("python3", verifier)


if __name__ == "__main__":
    unittest.main()
