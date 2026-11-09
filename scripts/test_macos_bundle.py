"""Verify signing order without using a developer certificate or player files."""
import tempfile
import unittest
from pathlib import Path
from macos_bundle import signing_targets


class BundleTests(unittest.TestCase):
    def test_nested_code_is_signed_before_containers(self):
        with tempfile.TemporaryDirectory() as directory:
            app = Path(directory).resolve() / 'Jornia.app'
            binary = app / 'Contents/MacOS/Jornia'
            helper = app / 'Contents/Frameworks/Capture.app'
            nested = helper / 'Contents/MacOS/Capture'
            for path in (binary, nested):
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(bytes.fromhex('cffaedfe') + b'fixture')
            (binary.parent / 'notes.txt').write_text('not executable code')
            targets = signing_targets(app)
            self.assertEqual(set(targets), {app, binary, helper, nested})
            self.assertLess(targets.index(nested), targets.index(helper))
            self.assertLess(targets.index(helper), targets.index(app))
            self.assertEqual(targets[-1], app)

    def test_external_symlink_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory).resolve()
            app = base / 'Jornia.app'
            app.mkdir()
            outside = base / 'private.txt'
            outside.write_text('fixture')
            (app / 'escape').symlink_to(outside)
            with self.assertRaisesRegex(ValueError, 'outside the app'):
                signing_targets(app)

    def test_internal_symlink_does_not_sign_same_code_twice(self):
        with tempfile.TemporaryDirectory() as directory:
            app = Path(directory).resolve() / 'Jornia.app'
            app.mkdir()
            binary = app / 'Capture'
            binary.write_bytes(bytes.fromhex('cafebabe'))
            (app / 'alias').symlink_to('Capture')
            self.assertEqual(signing_targets(app), [binary, app])


if __name__ == '__main__':
    unittest.main()
