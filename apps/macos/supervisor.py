"""First-party Mac lifecycle. Status-only JSON to Cocoa; private credentials stay local."""
import argparse
import fcntl
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request


def emit(state, message, **fields):
    print(json.dumps(dict(state=state, message=message, **fields)), flush=True)


def data_directory():
    base = Path.home() / 'Library/Application Support'
    return base / ('Wisp' if (base / 'Wisp/player.db').exists() else 'Jornia')


def stop_process(process, timeout):
    if process and process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            # Preserve potentially interrupted carriers for explicit recovery on next launch.
            os.killpg(process.pid, signal.SIGKILL)
            process.wait()
            return False
    return True


def retire_session(path):
    if not path.exists():
        return
    if path.is_symlink() or path.stat().st_mode & 0o077:
        raise ValueError('An unsafe session file needs local inspection; it was preserved.')
    session = json.loads(path.read_text())
    import re
    if not re.fullmatch(r'http://127\.0\.0\.1:\d+', session['base_url']):
        raise ValueError('An unexpected session file was preserved.')
    request = urllib.request.Request(session['base_url'] + '/local/connection', headers={'X-Wisp-Token': session['token'], 'Origin': session['base_url']})
    try:
        with urllib.request.urlopen(request, timeout=2):
            pass
    except urllib.error.URLError as error:
        # HTTP errors or timeouts can still mean a live owner. Only refused connections permit recovery.
        import errno
        if not isinstance(error.reason, OSError) or error.reason.errno != errno.ECONNREFUSED:
            raise ValueError('Another local companion may be active. Quit it before starting Jornia.') from None
    else:
        raise ValueError('Another Jornia companion is already running. Quit it before starting this app.')
    path.rename(path.with_name('ocr-session-interrupted-' + str(time.time_ns()) + '.json'))


def run(args):
    data = args.data or data_directory()
    data.mkdir(parents=True, exist_ok=True, mode=0o700)
    if args.action == 'install':
        from install_addon import install
        backup = install(args.resources / 'AddOns', args.root, data)
        emit('installed', 'Addon installed. Enable Jornia at your next normal game launch. Your saved history is preserved.')
        return
    stop = threading.Event()
    signal.signal(signal.SIGTERM, lambda *_: stop.set())
    signal.signal(signal.SIGINT, lambda *_: stop.set())
    backend = reader = None
    with (data / 'mac-app.lock').open('a') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise ValueError('Jornia is already running.') from None
        # Establish that no reader owns the carriers before inspecting or recovering a stale session.
        with (data / 'native-reader.lock').open('a') as reader_lock:
            try:
                fcntl.flock(reader_lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError:
                raise ValueError('Another native reader is active. Quit it before starting Jornia.') from None
            session_path = data / 'ocr-session.json'
            retire_session(session_path)
        required = [args.root / 'WTF', args.root / 'Interface/AddOns/Wisp/QRTransport.lua',
                    args.root / 'Interface/AddOns/WispOCRInbox080/Inbox.lua']
        if not all(p.exists() for p in required):
            raise ValueError('Choose your Forever folder and install the Jornia addon first.')
        # Use owner-only files; never relay child logs, questions or session tokens to the UI.
        log_path = data / 'native-app.log'
        if log_path.is_symlink():
            raise ValueError('A linked log file was preserved; local inspection is needed.')
        fd = os.open(log_path, os.O_CREAT | os.O_WRONLY | os.O_TRUNC, 0o600)
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, 'w') as log:
            try:
                emit('starting', 'Starting your local companion…')
                backend = subprocess.Popen([str(args.resources / 'bin/jornia'), '-no-browser', '-ocr-lab', '-listen', '127.0.0.1:0', '-data-dir', str(data)], stdout=log, stderr=log, start_new_session=True)
                deadline = time.monotonic() + 15
                while not session_path.exists() and backend.poll() is None and time.monotonic() < deadline and not stop.wait(.05):
                    pass
                if stop.is_set():
                    return
                if not session_path.exists() or backend.poll() is not None:
                    raise ValueError('The local companion could not start. Your data was preserved; see native-app.log in your local app data.')
                session = json.loads(session_path.read_text())
                emit('waiting', 'Companion running. Waiting for WoW and local screen-capture permission.', url=session['base_url'])
                command = [sys.executable, '-s', '-S', str(args.resources / 'scripts/companion/reader.py'), '--qr', '--root', str(args.root), '--session', str(session_path), '--capture', str(args.resources / 'bin/jornia-capture')]
                # Recovery is restricted to fixed owned files and takes the reader lock.
                recovery = subprocess.run(command + ['--recover'], stdout=log, stderr=log, timeout=10)
                if recovery.returncode:
                    raise ValueError('Interrupted addon files need inspection. Originals are preserved; no question was repeated.')
                reader = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=log, text=True, start_new_session=True)
                capture_failed = threading.Event()
                def watch():
                    for line in reader.stdout:
                        log.write(line); log.flush()
                        if line.startswith('Capture active.'):
                            emit('ready', 'Ready. Open Jornia in WoW and ask a question. Keep the chat visible while waiting.')
                        elif '"capture_detached"' in line:
                            emit('waiting', 'Waiting for the WoW window. Your local journey is still available.')
                        elif '"capture_error"' in line:
                            capture_failed.set()
                            emit('permission', 'Screen capture is unavailable. Use Allow local screen capture, then start again.')
                        elif '"delivery_not_confirmed"' in line or '"api_not_confirmed"' in line:
                            emit('attention', 'A reply could not be confirmed. Check the in-game draft; it will not be resent automatically.')
                        elif '"status": "delivered"' in line:
                            emit('ready', 'Reply delivered. Ready for your next question.')
                watcher = threading.Thread(target=watch, daemon=True)
                watcher.start()
                while not stop.wait(.25):
                    if os.getppid() == 1:
                        break
                    if backend.poll() is not None or reader.poll() is not None:
                        if reader.poll() is not None:
                            watcher.join(timeout=1)
                        if capture_failed.is_set():
                            raise ValueError('Allow Jornia in System Settings → Privacy & Security → Screen Recording, then start the connection again. Images stay on this Mac.')
                        raise ValueError('The chat connection stopped. Check screen-capture permission, then start again. Your journal is preserved.')
            finally:
                emit('stopping', 'Stopping the chat connection and restoring owned addon files…')
                clean = stop_process(reader, 22)
                stop_process(backend, 8)
                emit('stopped', 'Stopped. Your journey is saved locally.' if clean else 'Stopped. Interrupted addon files will be checked before the next start.')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=['start', 'install'])
    parser.add_argument('--resources', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--data', type=Path)
    args = parser.parse_args()
    try:
        run(args)
    except Exception as error:
        # Only our explicit user-facing failures are displayed. Other exceptions are not dumped.
        emit('error', str(error) if isinstance(error, ValueError) else 'Jornia could not continue. Your files were preserved; check the local app log.')
        sys.exit(1)
