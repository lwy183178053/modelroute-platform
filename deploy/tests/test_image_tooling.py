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


class ImageToolingContractTest(unittest.TestCase):
    def test_production_dependencies_are_pinned_by_digest(self):
        config = compose_config()
        self.assertEqual(
            config["services"]["postgres"]["image"],
            "postgres:17-alpine@sha256:742f40ea20b9ff2ff31db5458d127452988a2164df9e17441e191f3b72252193",
        )
        self.assertEqual(
            config["services"]["redis"]["image"],
            "redis:8-alpine@sha256:9d317178eceac8454a2284a9e6df2466b93c745529947f0cd42a0fa9609d7005",
        )

    def test_dockerfiles_have_official_pinned_defaults(self):
        api = (ROOT / "apps" / "api" / "Dockerfile").read_text(encoding="utf-8")
        gateway = (ROOT / "deploy" / "gateway" / "Dockerfile").read_text(
            encoding="utf-8"
        )
        self.assertIn("ARG GO_BUILDER_IMAGE=golang:1.24-alpine@sha256:", api)
        self.assertIn("FROM ${GO_BUILDER_IMAGE} AS builder", api)
        self.assertIn("ARG ALPINE_RUNTIME_IMAGE=alpine:3.22@sha256:", api)
        self.assertIn("FROM ${ALPINE_RUNTIME_IMAGE}", api)
        self.assertIn("ARG NODE_BUILDER_IMAGE=node:24-alpine@sha256:", gateway)
        self.assertIn("FROM ${NODE_BUILDER_IMAGE} AS web-builder", gateway)
        self.assertIn("ARG NGINX_RUNTIME_IMAGE=nginx:1.28-alpine@sha256:", gateway)
        self.assertIn("FROM ${NGINX_RUNTIME_IMAGE}", gateway)

    def test_local_environment_uses_daocloud_with_same_digests(self):
        dev_script = (ROOT / "deploy" / "scripts" / "dev.ps1").read_text(
            encoding="utf-8"
        )
        for digest in (
            "8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191",
            "14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce",
            "a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd",
            "a8b39bd9cf0f83869a2162827a0caf6137ddf759d50a171451b335cecc87d236",
            "742f40ea20b9ff2ff31db5458d127452988a2164df9e17441e191f3b72252193",
            "9d317178eceac8454a2284a9e6df2466b93c745529947f0cd42a0fa9609d7005",
            "428018a37c0b26c163a3367c18401161707cd0e08d0f26a3dde9ff0caa05e34c",
        ):
            self.assertIn(f"m.daocloud.io/docker.io/", dev_script)
            self.assertIn(digest, dev_script)

    def test_release_build_passes_daocloud_base_images(self):
        release = (ROOT / "deploy" / "scripts" / "release.ps1").read_text(
            encoding="utf-8"
        )
        self.assertIn("GO_BUILDER_IMAGE=m.daocloud.io/docker.io/library/golang", release)
        self.assertIn("ALPINE_RUNTIME_IMAGE=m.daocloud.io/docker.io/library/alpine", release)
        self.assertIn("NODE_BUILDER_IMAGE=m.daocloud.io/docker.io/library/node", release)
        self.assertIn("NGINX_RUNTIME_IMAGE=m.daocloud.io/docker.io/library/nginx", release)

    def test_preflight_waits_only_for_long_running_services(self):
        dev_script = (ROOT / "deploy" / "scripts" / "dev.ps1").read_text(
            encoding="utf-8"
        )
        self.assertIn(
            "Local-Compose @('up', '-d', '--wait', 'postgres', 'redis', 'new-api', "
            "'platform-api', 'platform-worker', 'gateway')",
            dev_script,
        )


if __name__ == "__main__":
    unittest.main()
