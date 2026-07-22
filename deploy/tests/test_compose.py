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


if __name__ == "__main__":
    unittest.main()
