import json
import subprocess
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def compose_config():
    result = subprocess.run(
        [
            "docker",
            "compose",
            "--env-file",
            ".env.example",
            "--profile",
            "tools",
            "-f",
            "compose.yaml",
            "config",
            "--format",
            "json",
        ],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    return json.loads(result.stdout)


class ComposeIsolationTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.config = compose_config()

    def test_gateway_has_loopback_port_and_non_internal_network(self):
        gateway = self.config["services"]["gateway"]
        self.assertEqual(
            gateway["ports"],
            [
                {
                    "mode": "ingress",
                    "host_ip": "127.0.0.1",
                    "target": 8080,
                    "published": "28100",
                    "protocol": "tcp",
                }
            ],
        )
        self.assertTrue(
            any(
                not self.config["networks"][name].get("internal", False)
                for name in gateway["networks"]
            ),
            "网关必须连接一个非内部网络，Docker 才会建立主机端口转发",
        )

    def test_datastores_remain_internal_only(self):
        self.assertEqual(list(self.config["services"]["postgres"]["networks"]), ["backend"])
        self.assertEqual(list(self.config["services"]["redis"]["networks"]), ["backend"])
        self.assertTrue(self.config["networks"]["backend"]["internal"])

    def test_new_api_image_is_pinned_by_digest(self):
        image = self.config["services"]["new-api"]["image"]
        self.assertIn("v1.0.0-rc.21@sha256:428018a3", image)

    def test_gateway_healthcheck_uses_ipv4_loopback(self):
        healthcheck = self.config["services"]["gateway"]["healthcheck"]["test"]
        self.assertIn("http://127.0.0.1:8080/healthz", healthcheck[1])

    def test_gateway_supports_versionless_openai_paths(self):
        gateway = (ROOT / "deploy" / "gateway" / "default.conf").read_text(encoding="utf-8")
        verifier = (ROOT / "deploy" / "scripts" / "verify.sh").read_text(encoding="utf-8")
        self.assertIn("responses|chat/completions|images/generations|models|usage", gateway)
        self.assertIn("rewrite ^ /v1$uri break;", gateway)
        self.assertIn('"$BASE_URL/responses"', verifier)
        self.assertIn('"code":"invalid_api_key"', verifier)

    def test_platform_services_receive_credential_encryption_key(self):
        for service_name in ("platform-api", "platform-worker"):
            environment = self.config["services"][service_name]["environment"]
            self.assertEqual(
                environment["PLATFORM_ENCRYPTION_KEY"],
                "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=",
            )
            self.assertEqual(environment["NEW_API_ADMIN_USERNAME"], "modelroute")
            self.assertEqual(environment["NEW_API_ADMIN_PASSWORD"], "CHANGE_TO_SERVER_ADMIN_PASSWORD")

    def test_model_seed_is_an_explicit_one_shot_tool(self):
        seed = self.config["services"]["model-seed"]
        self.assertEqual(seed["entrypoint"], ["/usr/local/bin/modelroute-seed-models"])
        self.assertEqual(seed["profiles"], ["tools"])
        self.assertEqual(seed["restart"], "no")
        self.assertEqual(seed["depends_on"]["postgres"]["condition"], "service_healthy")
        self.assertEqual(seed["depends_on"]["new-api"]["condition"], "service_healthy")

    def test_base_verifier_requires_api_key_from_enabled_model_route(self):
        verifier = (ROOT / "deploy" / "scripts" / "verify.sh").read_text(encoding="utf-8")
        self.assertIn('test "$status" = "401"', verifier)
        self.assertIn('"code":"invalid_api_key"', verifier)
        self.assertNotIn('"code":"service_not_ready"', verifier)

    def test_payment_verifier_covers_auth_and_invalid_callback(self):
        verifier = (ROOT / "deploy" / "scripts" / "verify-payments.sh").read_text(encoding="utf-8")
        self.assertIn("/api/payment-settings", verifier)
        self.assertIn("/api/admin/payment-settings", verifier)
        self.assertIn("/api/payments/easypay/notify", verifier)
        self.assertIn('test "$status" = "400"', verifier)
        self.assertNotIn("merchant_key", verifier)

    def test_withdrawal_verifier_is_read_only_and_covers_permissions(self):
        verifier = (ROOT / "deploy" / "scripts" / "verify-withdrawals.sh").read_text(encoding="utf-8")
        self.assertIn("/api/admin/withdrawals", verifier)
        self.assertIn("/api/merchant/balance", verifier)
        self.assertIn('test "$status" = "401"', verifier)
        self.assertIn('test "$status" = "403"', verifier)
        self.assertNotIn("settlement-reconciliation", verifier)
        self.assertNotIn("/ignore", verifier)
        self.assertNotIn("/paid", verifier)


if __name__ == "__main__":
    unittest.main()
