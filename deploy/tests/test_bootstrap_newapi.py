import importlib.util
import json
import tempfile
import unittest
from pathlib import Path


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


if __name__ == "__main__":
    unittest.main()
