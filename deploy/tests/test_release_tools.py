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
    def test_merchant_verifier_uses_temporary_common_user(self):
        verifier = read_script("verify-merchants.sh")
        for marker in (
            "ADMIN_COOKIE_JAR=",
            "USER_COOKIE_JAR=",
            "NEW_API_COOKIE_JAR=",
            "NEW_API_BASE_URL=",
            '"role": 1',
            'me.get("is_platform_merchant") is False',
            '-b "$USER_COOKIE_JAR"',
            '"$NEW_API_BASE_URL/api/user/"',
            '-X DELETE "$NEW_API_BASE_URL/api/user/$TEST_USER_ID"',
            "/api/merchant/channel-tests",
            '"canonical_model_id"',
            '"test_receipt"',
            "merchant_channel_tests",
            "receipt_token_hash",
            "PLATFORM_ENCRYPTION_KEY",
        ):
            self.assertIn(marker, verifier)
        for old_field in ('"channel_type"', '"models"'):
            self.assertNotIn(old_field, verifier)

    def test_production_verifiers_use_the_unique_admin_username(self):
        for name in (
            "verify-accounts.sh",
            "verify-merchants.sh",
            "verify-routing.sh",
            "verify-payments.sh",
            "verify-settlement.sh",
            "verify-operations.sh",
        ):
            self.assertIn(
                'ADMIN_USERNAME="${ADMIN_USERNAME:-3180615598@qq.com}"',
                read_script(name),
                name,
            )

    def test_routing_verifier_uses_the_v2_zero_to_five_range(self):
        verifier = read_script("verify-routing.sh")
        for marker in (
            'assert int(policy["minimum_multiplier_bps"]) == 0',
            'assert int(policy["maximum_multiplier_bps"]) == 50000',
            '"minimum_multiplier_bps":0',
            '"maximum_multiplier_bps":50000',
        ):
            self.assertIn(marker, verifier)
        for old_marker in (
            'assert int(policy["minimum_multiplier_bps"]) == 1',
            'assert int(policy["maximum_multiplier_bps"]) == 10000',
            '"minimum_multiplier_bps":1',
            '"maximum_multiplier_bps":10000',
        ):
            self.assertNotIn(old_marker, verifier)

    def test_official_model_seed_catalog_is_included_in_release(self):
        catalog = ROOT / "apps" / "api" / "internal" / "modelseed" / "data" / "2026-07-23-v2.json"
        self.assertTrue(catalog.is_file())
        result = subprocess.run(
            ["git", "check-ignore", "--quiet", str(catalog.relative_to(ROOT))],
            cwd=ROOT,
        )
        self.assertEqual(result.returncode, 1, "官方模型种子不得被 .gitignore 排除")

    def test_operations_verifier_is_read_only_and_covers_phase_eight(self):
        verifier = read_script("verify-operations.sh")
        for path in (
            "/api/usage",
            "/api/admin/overview",
            "/api/admin/usage",
            "/api/admin/platform-settings",
            "/api/admin/redemptions",
            "/api/admin/system-tasks",
            "/api/admin/channels",
        ):
            self.assertIn(path, verifier)
        self.assertNotIn("/api/admin/groups", verifier)
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

    def test_v2_backup_includes_redis_newapi_and_restricted_configuration(self):
        script = read_script("backup.sh")
        for marker in (
            "umask 077",
            "redis-cli SAVE",
            "redis.rdb",
            "newapi-data.tar.gz",
            "newapi-smtp-options.json",
            "env.backup",
            "bootstrap-admin.txt",
            "sha256sum",
        ):
            self.assertIn(marker, script)

    def test_v2_reset_is_confirmed_scoped_backed_up_and_bootstrapped(self):
        script = read_script("reset-v2-data.sh")
        for marker in (
            "CONFIRM_V2_RESET=MODELROUTE_V2_RESET_103",
            "38.34.191.103",
            "/opt/modelroute-platform/shared/data",
            'test "$COMPOSE_PROJECT_NAME" = "modelroute"',
            "backup.sh",
            "verify-backup-restore.sh",
            "sha256sum -c",
            "docker compose --env-file .env down",
            "bootstrap-newapi.py",
            "model-seed",
        ):
            self.assertIn(marker, script)
        for forbidden in ("/opt/sub2api", "docker system prune", "docker volume prune"):
            self.assertNotIn(forbidden, script)

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
