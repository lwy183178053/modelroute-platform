import os
import shlex
import shutil
import subprocess
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
            subprocess.run(["bash", "-lc", command], check=True)

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


if __name__ == "__main__":
    unittest.main()
