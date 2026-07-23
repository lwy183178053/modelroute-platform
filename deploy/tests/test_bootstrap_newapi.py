import importlib.util
import json
import os
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "deploy" / "scripts" / "bootstrap-newapi.py"
SPEC = importlib.util.spec_from_file_location("bootstrap_newapi", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class BootstrapNewAPITest(unittest.TestCase):
    def test_smtp_restore_accepts_only_approved_keys(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "smtp.json"
            path.write_text(
                json.dumps({"SMTPServer": "smtp.example.test", "SMTPPort": "465"}),
                encoding="utf-8",
            )
            self.assertEqual(
                MODULE.load_smtp_options(path),
                {"SMTPServer": "smtp.example.test", "SMTPPort": "465"},
            )

            path.write_text(
                json.dumps({"SMTPServer": "smtp.example.test", "Unexpected": "value"}),
                encoding="utf-8",
            )
            with self.assertRaisesRegex(RuntimeError, "未批准"):
                MODULE.load_smtp_options(path)

    def test_long_admin_username_uses_short_setup_then_renames_and_reauthenticates(self):
        target_username = "3180615598@qq.com"
        calls = []

        def fake_request(opener, base_url, method, path, payload=None, headers=None):
            calls.append((method, path, payload, headers))
            if path == "/api/setup":
                self.assertLessEqual(len(payload["username"]), 12)
                self.assertNotEqual(payload["username"], target_username)
                return None
            if path == "/api/user/login":
                return {
                    "id": 1,
                    "username": payload["username"],
                    "role": 100,
                    "status": 1,
                    "group": "default",
                }
            if path == "/api/user/":
                self.assertEqual(payload["id"], 1)
                self.assertEqual(payload["username"], target_username)
                return None
            if path == "/api/option/":
                self.assertEqual(headers, {"New-Api-User": "1"})
                return None
            self.fail(f"未预期的请求：{method} {path}")

        with (
            mock.patch.object(MODULE, "wait_ready", return_value={"status": False}),
            mock.patch.object(MODULE, "request", side_effect=fake_request),
            mock.patch.object(
                MODULE,
                "load_smtp_options",
                return_value={"SMTPServer": "smtp.example.test"},
            ),
            mock.patch.object(MODULE.urllib.request, "build_opener", return_value=object()),
            mock.patch.object(
                sys,
                "argv",
                ["bootstrap-newapi.py", "http://new-api:3000", "smtp.json"],
            ),
            mock.patch.dict(
                os.environ,
                {
                    "NEW_API_ADMIN_USERNAME": target_username,
                    "NEW_API_ADMIN_PASSWORD": "Complex-password-1",
                },
            ),
        ):
            MODULE.main()

        self.assertEqual(
            [(method, path) for method, path, _, _ in calls],
            [
                ("POST", "/api/setup"),
                ("POST", "/api/user/login"),
                ("PUT", "/api/user/"),
                ("POST", "/api/user/login"),
                ("PUT", "/api/option/"),
            ],
        )
        self.assertEqual(calls[1][2]["username"], calls[0][2]["username"])
        self.assertEqual(calls[3][2]["username"], target_username)

    def test_initialized_temporary_admin_is_resumed_and_renamed(self):
        target_username = "3180615598@qq.com"
        calls = []
        target_login_attempts = 0

        def fake_request(opener, base_url, method, path, payload=None, headers=None):
            nonlocal target_login_attempts
            calls.append((method, path, payload, headers))
            if path == "/api/user/login" and payload["username"] == target_username:
                target_login_attempts += 1
                if target_login_attempts == 1:
                    raise RuntimeError("用户名或密码错误")
            if path == "/api/user/login":
                return {
                    "id": 1,
                    "username": payload["username"],
                    "role": 100,
                    "status": 1,
                    "group": "default",
                }
            if path == "/api/user/":
                self.assertEqual(payload["username"], target_username)
                return None
            self.fail(f"未预期的请求：{method} {path}")

        with (
            mock.patch.object(MODULE, "wait_ready", return_value={"status": True}),
            mock.patch.object(MODULE, "request", side_effect=fake_request),
            mock.patch.object(MODULE, "load_smtp_options", return_value={}),
            mock.patch.object(MODULE.urllib.request, "build_opener", return_value=object()),
            mock.patch.object(
                sys,
                "argv",
                ["bootstrap-newapi.py", "http://new-api:3000", "smtp.json"],
            ),
            mock.patch.dict(
                os.environ,
                {
                    "NEW_API_ADMIN_USERNAME": target_username,
                    "NEW_API_ADMIN_PASSWORD": "Complex-password-1",
                },
            ),
        ):
            try:
                MODULE.main()
            except RuntimeError as error:
                self.fail(f"脚本没有从临时管理员中断点恢复：{error}")

        self.assertEqual(
            [(method, path) for method, path, _, _ in calls],
            [
                ("POST", "/api/user/login"),
                ("POST", "/api/user/login"),
                ("PUT", "/api/user/"),
                ("POST", "/api/user/login"),
            ],
        )


if __name__ == "__main__":
    unittest.main()
