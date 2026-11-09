"""Bundle the builder's CPython and relocate its native dependencies inside the app."""
import os
from pathlib import Path
import shutil
import subprocess
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from macos_bundle import MACH_O


def native(path):
    if not path.is_file() or path.is_symlink():
        return False
    with path.open('rb') as source:
        return source.read(4) in MACH_O


def dependencies(path):
    output = subprocess.check_output(['otool', '-L', str(path)], text=True)
    return [line.strip().split(' (')[0] for line in output.splitlines()[1:] if ' (' in line]


def bundle(destination):
    destination = Path(destination)
    base = Path(sys.base_prefix).resolve()
    version = f'{sys.version_info.major}.{sys.version_info.minor}'
    framework = base / 'Python'
    if not framework.is_file():
        raise ValueError('Build with a macOS framework CPython; the player will not need it installed')
    (destination / 'bin').mkdir(parents=True)
    (destination / 'lib').mkdir()
    # Framework bin/python is a trampoline to Resources/Python.app. Bundle the
    # actual interpreter so it cannot spawn an executable outside our private runtime.
    interpreter = base / 'Resources/Python.app/Contents/MacOS/Python'
    if not interpreter.is_file():
        raise ValueError('The framework interpreter executable is missing')
    shutil.copy2(interpreter, destination / 'bin/python3')
    shutil.copy2(framework, destination / 'lib/Python')
    shutil.copytree(base / 'lib' / ('python' + version), destination / 'lib' / ('python' + version),
                    ignore=shutil.ignore_patterns('__pycache__', '*.pyc', 'site-packages', 'test', 'tests', 'idlelib', 'tkinter', 'ensurepip', '._*'))
    originals = {str(framework): destination / 'lib/Python'}
    copied = {framework.resolve(): destination / 'lib/Python'}
    # Homebrew uses both /opt/.../opt and /opt/.../Cellar spellings.
    for path in [destination / 'bin/python3', destination / 'lib/Python']:
        for dep in dependencies(path):
            if Path(dep).name == 'Python':
                originals[dep] = destination / 'lib/Python'
    pending = [path for path in destination.rglob('*') if native(path)]
    seen = set()
    while pending:
        path = pending.pop()
        if path in seen:
            continue
        seen.add(path)
        ids = subprocess.check_output(['otool', '-D', str(path)], text=True).splitlines()[1:]
        for dep in dependencies(path):
            if dep in ids:
                continue
            if dep.startswith(('/usr/lib/', '/System/Library/')):
                continue
            if dep.startswith('@'):
                raise ValueError('Unresolved runtime dependency: ' + dep)
            if dep not in originals:
                source = Path(dep).resolve(strict=True)
                if source in copied:
                    originals[dep] = copied[source]
                else:
                    target = destination / 'lib' / source.name
                    if target.exists():
                        raise ValueError('Colliding runtime library names: ' + source.name)
                    shutil.copy2(source, target)
                    # Preserve licences for each linked Homebrew dependency, not just CPython.
                    if '/Cellar/' in str(source):
                        pieces = source.parts
                        cellar = pieces.index('Cellar')
                        package_root = Path(*pieces[:cellar + 3])
                        notices = destination / 'THIRD-PARTY' / (pieces[cellar + 1] + '-' + pieces[cellar + 2])
                        for pattern in ('LICENSE*', 'COPYING*', 'COPYRIGHT*', 'NOTICE*'):
                            for notice in package_root.rglob(pattern):
                                if notice.is_file() and not notice.name.startswith('._'):
                                    notices.mkdir(parents=True, exist_ok=True)
                                    shutil.copy2(notice, notices / ('_'.join(notice.relative_to(package_root).parts)))
                    copied[source] = target
                    originals[dep] = target
                    pending.append(target)
            relative = os.path.relpath(originals[dep], path.parent)
            subprocess.run(['install_name_tool', '-change', dep, '@loader_path/' + relative, str(path)], check=True, capture_output=True)
        if ids:
            subprocess.run(['install_name_tool', '-id', '@loader_path/' + path.name, str(path)], check=True, capture_output=True)
        # Remove copied signatures before a final inside-out signing pass.
        subprocess.run(['codesign', '--remove-signature', str(path)], capture_output=True)
    # Extension modules discover the standard library through PYTHONHOME; no external site packages.
    return version
