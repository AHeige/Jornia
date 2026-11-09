"""Exercise install and session ownership with disposable folders, never the player's game."""
import errno
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import urllib.error

from install_addon import install
from supervisor import retire_session


class LifecycleTests(unittest.TestCase):
    def fixture(self, base):
        root, data, source = base / 'game', base / 'data', base / 'source'
        (root / 'WTF/Account').mkdir(parents=True)
        (root / 'WTF/Account/saved.lua').write_text('private fixture')
        folder = source / 'Wisp'
        folder.mkdir(parents=True)
        (folder / 'Wisp_Camelot.toc').write_text('## Title: Jornia\n')
        (folder / 'State.lua').write_text('new fixture')
        return source, root, data

    def test_install_backups_preserve_existing_addon_and_journal(self):
        with tempfile.TemporaryDirectory() as directory:
            source, root, data = self.fixture(Path(directory))
            target = root / 'Interface/AddOns/Wisp'
            target.mkdir(parents=True)
            (target / 'Wisp_Camelot.toc').write_text('## Title: Wisp\n')
            (target / 'State.lua').write_text('old fixture')
            backup = install(source, root, data, game_running=lambda: False)
            self.assertEqual((backup / 'Wisp/State.lua').read_text(), 'old fixture')
            self.assertEqual((target / 'State.lua').read_text(), 'new fixture')
            self.assertEqual((root / 'WTF/Account/saved.lua').read_text(), 'private fixture')

    def test_running_game_does_not_get_modified(self):
        with tempfile.TemporaryDirectory() as directory:
            source, root, data = self.fixture(Path(directory))
            with self.assertRaisesRegex(ValueError, 'WoW is open'):
                install(source, root, data, game_running=lambda: True)
            self.assertFalse((root / 'Interface').exists())

    def test_interrupted_carrier_prevents_install(self):
        with tempfile.TemporaryDirectory() as directory:
            source, root, data = self.fixture(Path(directory))
            target = root / 'Interface/AddOns/Wisp'
            target.mkdir(parents=True)
            (target / 'owned.wispbus-hold').write_text('preserve')
            with self.assertRaisesRegex(ValueError, 'recovery'):
                install(source, root, data, game_running=lambda: False)
            self.assertEqual((target / 'owned.wispbus-hold').read_text(), 'preserve')

    def test_live_session_is_preserved(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'ocr-session.json'
            path.write_text(json.dumps(dict(base_url='http://127.0.0.1:1234', token='synthetic')))
            path.chmod(0o600)
            with patch('supervisor.urllib.request.urlopen'):
                with self.assertRaisesRegex(ValueError, 'already running'):
                    retire_session(path)
            self.assertTrue(path.exists())

    def test_only_refused_connection_allows_stale_session_retirement(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'ocr-session.json'
            path.write_text(json.dumps(dict(base_url='http://127.0.0.1:1234', token='synthetic')))
            path.chmod(0o600)
            with patch('supervisor.urllib.request.urlopen', side_effect=urllib.error.URLError(TimeoutError())):
                with self.assertRaisesRegex(ValueError, 'may be active'):
                    retire_session(path)
            self.assertTrue(path.exists())
            with patch('supervisor.urllib.request.urlopen', side_effect=urllib.error.URLError(ConnectionRefusedError(errno.ECONNREFUSED, 'fixture'))):
                retire_session(path)
            self.assertFalse(path.exists())
            self.assertEqual(len(list(path.parent.glob('ocr-session-interrupted-*.json'))), 1)


if __name__ == '__main__':
    unittest.main()
