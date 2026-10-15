"""Installer regression tests. Downloads use a local HTTP server, never GitHub."""
import contextlib
import hashlib
import http.server
import io
import json
import os
from pathlib import Path
import platform
import subprocess
import tarfile
import tempfile
import threading
import unittest

SCRIPT = Path(__file__).with_name("install.sh").resolve()


class QuietHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *_args):
        pass


class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.dist = self.root / "dist"
        self.dist.mkdir()
        self.install = self.root / "bin"
        self.install.mkdir()
        self.scratch = self.root / "tmp"
        self.scratch.mkdir()
        os_name = {"Darwin": "darwin", "Linux": "linux"}[platform.system()]
        arch = {"arm64": "arm64", "aarch64": "arm64", "x86_64": "amd64"}[platform.machine()]
        self.archive = f"memvault_1.2.3_{os_name}_{arch}.tar.gz"
        self.binary = self.install / "memvault"
        self.binary.write_text("old installation")
        self.env = dict(os.environ, VERSION="1.2.3", DIST_DIR=str(self.dist),
                        INSTALL_DIR=str(self.install), TMPDIR=str(self.scratch))

    def package(self, directory=None):
        directory = directory or self.dist
        directory.mkdir(parents=True, exist_ok=True)
        data = b"#!/bin/sh\necho 1.2.3\n"
        with tarfile.open(directory / self.archive, "w:gz") as archive:
            info = tarfile.TarInfo("memvault")
            info.size = len(data)
            info.mode = 0o755
            archive.addfile(info, io.BytesIO(data))
        digest = hashlib.sha256((directory / self.archive).read_bytes()).hexdigest()
        (directory / "SHA256SUMS").write_text(f"{digest}  {self.archive}\n")

    def run_install(self, **overrides):
        result = subprocess.run(["bash", str(SCRIPT)], env=dict(self.env, **overrides),
                                cwd=self.root, capture_output=True, text=True, timeout=15)
        self.assertEqual(list(self.scratch.iterdir()), [], "temporary files leaked")
        return result

    @contextlib.contextmanager
    def server(self):
        handler = lambda *args, **kwargs: QuietHandler(*args, directory=str(self.root), **kwargs)
        with http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler) as server:
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            try:
                yield f"http://127.0.0.1:{server.server_port}"
            finally:
                server.shutdown()
                thread.join()

    def assert_installed(self):
        self.assertEqual(subprocess.check_output([str(self.binary), "version"], text=True).strip(), "1.2.3")
        self.assertEqual(self.binary.with_suffix(".previous").read_text(), "old installation")

    def test_local_install_and_backup(self):
        self.package()
        result = self.run_install(VERSION="v1.2.3")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_installed()

    def test_corrupt_archive_preserves_existing_install(self):
        self.package()
        with (self.dist / self.archive).open("ab") as archive:
            archive.write(b"corrupted")
        result = self.run_install()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SHA-256 mismatch", result.stderr)
        self.assertEqual(self.binary.read_text(), "old installation")
        self.assertFalse(self.binary.with_suffix(".previous").exists())

    def test_missing_manifest_preserves_existing_install(self):
        self.package()
        (self.dist / "SHA256SUMS").unlink()
        result = self.run_install()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.binary.read_text(), "old installation")

    def test_missing_checksum_entry_is_rejected(self):
        self.package()
        (self.dist / "SHA256SUMS").write_text("0" * 64 + "  another-package.tar.gz\n")
        result = self.run_install()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.binary.read_text(), "old installation")

    def test_download_latest_release(self):
        self.package(self.root / "releases" / "v1.2.3")
        (self.root / "latest.json").write_text(json.dumps({"tag_name": "v1.2.3"}))
        with self.server() as url:
            result = self.run_install(VERSION="", BASE_URL=url + "/releases", API_URL=url + "/latest.json")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_installed()

    def test_failed_download_preserves_existing_install(self):
        with self.server() as url:
            result = self.run_install(BASE_URL=url + "/missing")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.binary.read_text(), "old installation")

    def test_invalid_version_is_rejected(self):
        result = self.run_install(VERSION="../../another-file")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Invalid version", result.stderr)
        self.assertEqual(self.binary.read_text(), "old installation")


if __name__ == "__main__":
    unittest.main(verbosity=2)
