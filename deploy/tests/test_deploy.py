import os
import hashlib
import shlex
import shutil
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def shell_path(path):
    if os.name != "nt":
        return str(path)
    drive, tail = os.path.splitdrive(str(Path(path).resolve()))
    tail = tail.lstrip("\\").replace("\\", "/")
    return f"/mnt/{drive[0].lower()}/{tail}"


class DeployScriptTest(unittest.TestCase):
    def test_release_directory_is_exported_as_app_version(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            release = workspace / "releases" / "release-abc123"
            current = workspace / "current"
            scripts = release / "deploy" / "scripts"
            fake_bin = release / "fake-bin"
            scripts.mkdir(parents=True)
            fake_bin.mkdir()
            shutil.copy2(ROOT / "deploy" / "scripts" / "deploy.sh", scripts / "deploy.sh")
            (release / ".env").write_text("APP_VERSION=stale\n", encoding="ascii")
            docker = fake_bin / "docker"
            docker.write_bytes(
                b'#!/bin/sh\nprintf "%s|%s\\n" "$APP_VERSION" "$*" >> "$DEPLOY_TEST_LOG"\ncase "$*" in *to_regclass*) printf "1\\n";; *string_agg*) printf "9,12\\n";; esac\n'
            )
            docker.chmod(0o755)

            command = " ".join(
                [
                    f"ln -s {shlex.quote(shell_path(release))} {shlex.quote(shell_path(current))}",
                    "&&",
                    f"PATH={shlex.quote(shell_path(fake_bin))}:/usr/local/bin:/usr/bin:/bin",
                    f"DEPLOY_TEST_LOG={shlex.quote(shell_path(release / 'docker.log'))}",
                    f"bash {shlex.quote(shell_path(current / 'deploy' / 'scripts' / 'deploy.sh'))}",
                ]
            )
            result = subprocess.run(
                ["bash", "-lc", command],
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

            calls = (release / "docker.log").read_text(encoding="ascii").splitlines()
            self.assertEqual([line.split("|", 1)[0] for line in calls], [release.name] * 9)
            self.assertEqual(
                [line.split("|", 1)[1] for line in calls],
                [
                    "compose --env-file .env --profile tools config --quiet",
                    "compose --env-file .env --profile tools build",
                    "compose --env-file .env up -d postgres redis new-api",
                    "compose --env-file .env exec -T postgres sh -c psql -At -U \"$POSTGRES_USER\" -d \"$PLATFORM_DB_NAME\" -c \"SELECT CASE WHEN to_regclass('public.platform_schema_migrations') IS NULL THEN 0 ELSE 1 END\"",
                    "compose --env-file .env exec -T postgres sh -c psql -At -U \"$POSTGRES_USER\" -d \"$PLATFORM_DB_NAME\" -c \"SELECT CASE WHEN EXISTS (SELECT 1 FROM platform_schema_migrations WHERE version = 11) THEN '__MIGRATED__' ELSE COALESCE((SELECT string_agg(newapi_token_id::text, ',') FROM token_bindings), '') END\"",
                    "compose --env-file .env exec -T postgres sh -c psql -v ON_ERROR_STOP=1 -U \"$POSTGRES_USER\" -d \"$NEWAPI_DB_NAME\" -c \"DELETE FROM tokens WHERE id IN (9,12)\"",
                    "compose --env-file .env run --rm model-seed",
                    "compose --env-file .env up -d --remove-orphans",
                    "compose --env-file .env ps",
                ],
            )

    def test_release_manifest_pulls_images_without_building(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            release = workspace / "0123456789ab"
            scripts = release / "deploy" / "scripts"
            fake_bin = release / "fake-bin"
            scripts.mkdir(parents=True)
            fake_bin.mkdir()
            shutil.copy2(ROOT / "deploy" / "scripts" / "deploy.sh", scripts / "deploy.sh")
            (release / ".env").write_text("APP_VERSION=stale\n", encoding="ascii")
            digest_a = "a" * 64
            digest_b = "b" * 64
            (release / "release-images.env").write_bytes(
                (
                    "\n".join(
                    [
                        "RELEASE_VERSION=0123456789ab",
                        f"PLATFORM_IMAGE=ghcr.io/lwy183178053/modelroute-platform-api@sha256:{digest_a}",
                        f"GATEWAY_IMAGE=ghcr.io/lwy183178053/modelroute-platform-gateway@sha256:{digest_b}",
                    ]
                    )
                    + "\n"
                ).encode("ascii")
            )
            docker = fake_bin / "docker"
            docker.write_bytes(
                b'#!/bin/sh\nprintf "%s|%s|%s|%s\\n" "$APP_VERSION" "$PLATFORM_IMAGE" "$GATEWAY_IMAGE" "$*" >> "$DEPLOY_TEST_LOG"\ncase "$*" in *to_regclass*) printf "1\\n";; *string_agg*) printf "__MIGRATED__\\n";; esac\n'
            )
            docker.chmod(0o755)

            command = " ".join(
                [
                    f"cd {shlex.quote(shell_path(release))}",
                    "&&",
                    f"PATH={shlex.quote(shell_path(fake_bin))}:/usr/local/bin:/usr/bin:/bin",
                    f"DEPLOY_TEST_LOG={shlex.quote(shell_path(release / 'docker.log'))}",
                    f"bash -x {shlex.quote(shell_path(scripts / 'deploy.sh'))}",
                ]
            )
            result = subprocess.run(
                ["bash", "-lc", command],
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

            calls = (release / "docker.log").read_text(encoding="ascii").splitlines()
            self.assertTrue(any("compose --env-file .env --profile tools pull" in line for line in calls))
            self.assertFalse(any(" build" in line for line in calls))
            self.assertTrue(all(line.startswith("0123456789ab|") for line in calls))
            self.assertTrue(any(f"@sha256:{digest_a}" in line for line in calls))
            self.assertTrue(any(f"@sha256:{digest_b}" in line for line in calls))

    def test_failed_release_restores_previous_pointer_and_service(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            base = workspace / "modelroute-platform"
            releases = base / "releases"
            shared = base / "shared"
            previous = releases / "aaaaaaaaaaaa"
            (shared / "data").mkdir(parents=True)
            (shared / "backups").mkdir()
            (shared / ".env").write_text("TEST=1\n", encoding="ascii")
            (shared / "bootstrap-admin.txt").write_text("restricted\n", encoding="ascii")
            (previous / "deploy" / "scripts").mkdir(parents=True)
            previous_deploy = previous / "deploy" / "scripts" / "deploy.sh"
            previous_deploy.write_bytes(
                b'#!/bin/sh\nprintf "restored\\n" >> "$MODELROUTE_RELEASE_BASE/restore.log"\n'
            )
            previous_deploy.chmod(0o755)
            subprocess.run(
                [
                    "bash",
                    "-lc",
                    f"ln -s {shlex.quote(shell_path(previous))} {shlex.quote(shell_path(base / 'current'))}",
                ],
                check=True,
            )

            source = workspace / "source"
            scripts = source / "deploy" / "scripts"
            scripts.mkdir(parents=True)
            failing_deploy = scripts / "deploy.sh"
            failing_deploy.write_bytes(b"#!/bin/sh\nexit 19\n")
            failing_deploy.chmod(0o755)
            archive = workspace / "release.tar"
            with tarfile.open(archive, "w") as bundle:
                for item in source.rglob("*"):
                    bundle.add(item, arcname=item.relative_to(source))
            manifest = workspace / "release.env"
            manifest.write_bytes(
                (
                    "RELEASE_VERSION=bbbbbbbbbbbb\n"
                    f"PLATFORM_IMAGE=ghcr.io/lwy183178053/modelroute-platform-api@sha256:{'a' * 64}\n"
                    f"GATEWAY_IMAGE=ghcr.io/lwy183178053/modelroute-platform-gateway@sha256:{'b' * 64}\n"
                ).encode("ascii")
            )
            checksum = hashlib.sha256(archive.read_bytes()).hexdigest()

            command = " ".join(
                [
                    f"MODELROUTE_RELEASE_BASE={shlex.quote(shell_path(base))}",
                    "MODELROUTE_RELEASE_TESTING=1",
                    "bash -x",
                    shlex.quote(shell_path(ROOT / "deploy" / "scripts" / "install-release.sh")),
                    "bbbbbbbbbbbb",
                    shlex.quote(shell_path(archive)),
                    shlex.quote(shell_path(manifest)),
                    checksum,
                ]
            )
            result = subprocess.run(
                ["bash", "-lc", command],
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
            )

            self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
            current_target = subprocess.run(
                ["bash", "-lc", f"readlink -f {shlex.quote(shell_path(base / 'current'))}"],
                check=True,
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
            ).stdout.strip()
            self.assertEqual(current_target, shell_path(previous))
            self.assertTrue(
                (base / "restore.log").is_file(), result.stdout + result.stderr
            )
            self.assertEqual((base / "restore.log").read_text(encoding="ascii"), "restored\n")
            self.assertFalse((releases / "bbbbbbbbbbbb").exists())
            self.assertTrue((shared / "data").is_dir())
            self.assertTrue((shared / "backups").is_dir())


if __name__ == "__main__":
    unittest.main()
