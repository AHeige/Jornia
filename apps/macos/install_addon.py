"""Install only bundled, owned addon files while WoW and the native reader are stopped."""
import datetime
import fcntl
from pathlib import Path
import shutil
import subprocess
import tempfile


def running():
    return subprocess.run(['pgrep', '-x', 'World of Warcraft'], stdout=subprocess.DEVNULL).returncode == 0


def install(source, root, data, game_running=running):
    data = Path(data)
    data.mkdir(parents=True, exist_ok=True)
    with (data / 'mac-app.lock').open('a') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise ValueError('Stop the chat connection before updating the addon.') from None
        return _install(source, root, data, game_running)


def _install(source, root, data, game_running):
    source, root, data = Path(source), Path(root), Path(data)
    if not (root / 'WTF').is_dir() or root.is_symlink():
        raise ValueError('Choose the Forever folder containing WTF and Interface.')
    if game_running():
        raise ValueError('WoW is open. Install the addon before your next normal game launch; no files were changed.')
    data.mkdir(parents=True, exist_ok=True)
    with (data / 'native-reader.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        addons = root / 'Interface/AddOns'
        if any(p.is_symlink() for p in [root / 'Interface', addons]):
            raise ValueError('Linked addon folders are preserved; choose a direct game folder.')
        addons.mkdir(parents=True, exist_ok=True)
        folders = list(source.iterdir())
        for folder in folders:
            target = addons / folder.name
            if target.is_symlink() or (target.exists() and not target.is_dir()):
                raise ValueError('Unexpected addon path; original preserved.')
            if target.exists():
                if any(p.is_symlink() or p.name.endswith(('.wispbus-hold', '.wispconnect-hold')) for p in target.rglob('*')):
                    raise ValueError('An addon needs recovery first; originals preserved.')
                toc = target / (folder.name + '_Camelot.toc')
                if not toc.is_file() or not any(marker in toc.read_text() for marker in ('Jornia', 'Wisp')):
                    raise ValueError('Unrecognized existing addon; originals preserved.')
        backup = Path(tempfile.mkdtemp(prefix=datetime.datetime.now().strftime('addons-%Y%m%d-%H%M%S-'), dir=data))
        for folder in folders:
            target = addons / folder.name
            if target.exists():
                shutil.copytree(target, backup / folder.name)
        changed = []
        try:
            for folder in folders:
                target = addons / folder.name
                changed.append(target)
                shutil.copytree(folder, target, dirs_exist_ok=True)
        except Exception:
            for target in reversed(changed):
                shutil.rmtree(target)
                if (backup / target.name).exists():
                    shutil.copytree(backup / target.name, target)
            raise
        return backup
