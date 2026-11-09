#!/usr/bin/env python3
"""Exercise a packaged runtime with disposable game/data folders and a synthetic capture helper."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import queue
import shutil
import signal
import subprocess
import tempfile
import threading
import time


def check(app, denied):
    resources = app.resolve() / 'Contents/Resources'
    with tempfile.TemporaryDirectory(prefix='jornia-native-smoke-') as directory:
        base = Path(directory)
        root, data, fixture = base / 'game', base / 'data', base / 'Resources'
        (root / 'WTF').mkdir(parents=True)
        shutil.copytree(resources / 'AddOns', root / 'Interface/AddOns', ignore=shutil.ignore_patterns('._*'))
        (fixture / 'bin').mkdir(parents=True)
        (fixture / 'bin/jornia').symlink_to(resources / 'bin/jornia')
        (fixture / 'scripts').symlink_to(resources / 'scripts', target_is_directory=True)
        helper = fixture / 'bin/jornia-capture'
        helper.write_text("#!/bin/sh\nprintf '%s\\n' '{\"error\":\"screen_permission_required\"}'\nexit 3\n" if denied else "#!/bin/sh\nprintf '%s\\n' '{\"ready\":true}' '{\"capture_tick\":true}'\nwhile :; do sleep 1; done\n")
        helper.chmod(0o755)
        before = {p.relative_to(root): hashlib.sha256(p.read_bytes()).hexdigest() for p in root.rglob('*') if p.is_file()}
        env = {'HOME': str(base), 'PATH': '/usr/bin:/bin', 'PYTHONHOME': str(resources / 'Runtime'), 'PYTHONNOUSERSITE': '1', 'PYTHONDONTWRITEBYTECODE': '1', 'PYTHONUNBUFFERED': '1', 'SSL_CERT_FILE': str(resources / 'cacert.pem')}
        process = subprocess.Popen([str(resources / 'Runtime/bin/python3'), '-s', '-S', str(resources / 'macos/supervisor.py'), 'start', '--resources', str(fixture), '--root', str(root), '--data', str(data)], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=env)
        events = queue.Queue()
        def consume():
            for line in process.stdout:
                events.put(json.loads(line))
        thread = threading.Thread(target=consume, daemon=True)
        thread.start()
        states = []
        deadline = time.monotonic() + 20
        try:
            while time.monotonic() < deadline:
                try:
                    event = events.get(timeout=.2)
                except queue.Empty:
                    if process.poll() is not None:
                        break
                    continue
                states.append(event['state'])
                if event['state'] == 'ready':
                    process.send_signal(signal.SIGTERM)
                    break
            process.wait(timeout=15)
            thread.join(timeout=2)
            while not events.empty():
                states.append(events.get()['state'])
            if denied:
                assert 'permission' in states and 'error' in states and process.returncode == 1, states
            else:
                assert 'ready' in states and 'stopped' in states and process.returncode == 0, states
            after = {p.relative_to(root): hashlib.sha256(p.read_bytes()).hexdigest() for p in root.rglob('*') if p.is_file()}
            assert before == after, 'Owned addon files were not restored exactly'
            assert (data / 'player.db').exists() and not (data / 'ocr-session.json').exists()
            print(('Denied-permission recovery' if denied else 'Start/ready/graceful quit') + ': PASS; addon hashes restored, database retained, session removed')
        finally:
            if process.poll() is None:
                process.terminate()
                process.wait(timeout=30)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('app', type=Path)
    args = parser.parse_args()
    check(args.app, False)
    check(args.app, True)
