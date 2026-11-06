import pathlib
import tempfile
import unittest
from unittest.mock import patch
from pulse_return_presence import pulse


class ReturnPresenceTest(unittest.TestCase):
    def prepare(self, root):
        (root / 'WTF').mkdir()
        lab = root / 'Interface/AddOns/WispBridgeLab'
        lab.mkdir(parents=True)
        target = lab / 'wispreturn_pulse.wav'
        target.write_bytes(b'')
        return target

    def test_interrupt_restores_own_asset(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            target = self.prepare(root)
            with patch('pulse_return_presence.time.sleep', side_effect=KeyboardInterrupt):
                with self.assertRaises(KeyboardInterrupt):
                    pulse(root, 10, .5)
            self.assertEqual(target.read_bytes(), b'')

    def test_refuses_nonempty_and_symlink_without_changes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            target = self.prepare(root)
            target.write_bytes(b'preserve')
            with self.assertRaises(ValueError): pulse(root, 1, .5)
            self.assertEqual(target.read_bytes(), b'preserve')
            target.unlink()
            other = root / 'other';other.write_bytes(b'')
            target.symlink_to(other)
            with self.assertRaises(ValueError): pulse(root, 1, .5)
            self.assertTrue(target.is_symlink())
            self.assertTrue(other.exists())


if __name__ == '__main__': unittest.main()
