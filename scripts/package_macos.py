#!/usr/bin/env python3
"""Build a self-contained native Mac candidate, without altering published downloads."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import plistlib
import re
import shutil
import ssl
import subprocess
import sys
import tempfile

from macos.bundle_runtime import bundle, dependencies, native
from macos_bundle import signing_targets

ROOT = Path(__file__).resolve().parents[1]


def run(command, **kwargs):
    subprocess.run([str(part) for part in command], check=True, **kwargs)


def prepare_addons(destination):
    addon = destination / 'Wisp'
    shutil.copytree(ROOT / 'apps/addon/Jornia', addon, ignore=shutil.ignore_patterns('._*', '.DS_Store'))
    toc = addon / 'Wisp_Camelot.toc'
    lines = toc.read_text().splitlines()
    lines = ['## Version: 0.9.0-native-pilot' if line.startswith('## Version:') else line for line in lines if line not in ('QR.lua', 'QRTransport.lua')]
    index = lines.index('Transport.lua')
    lines[index:index] = ['QR.lua', 'QRTransport.lua']
    toc.write_text('\n'.join(lines) + '\n')
    # Compatibility names retained by the verified native protocol; no old experiment UI is loaded.
    marker = destination / 'WispOCR'
    marker.mkdir()
    (marker / 'WispOCR_Camelot.toc').write_text('## Interface: 16001\n## Title: Jornia native transport\n## Notes: Owned compatibility marker; no experimental chat UI.\n')
    bridge = destination / 'WispBridgeLab'
    bridge.mkdir()
    (bridge / 'WispBridgeLab_Camelot.toc').write_text('## Interface: 16001\n## Title: Jornia local connection\n')
    for filename in ('wispreturn_present.wav', 'wispreturn_pulse.wav'):
        (bridge / filename).write_bytes(b'')
    for n in range(1, 81):
        name = f'WispOCRInbox{n:03d}'
        folder = destination / name
        folder.mkdir()
        (folder / (name + '_Camelot.toc')).write_text('## Interface: 16001\n## Title: Jornia native connection\n## LoadOnDemand: 1\n## Dependencies: WispOCR\nInbox.lua\n')
        (folder / 'Inbox.lua').write_text('-- Owned native transport asset. No finite reply inbox is used.\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, default=ROOT / 'dist/jornia-0.9.0-mac-candidate')
    args = parser.parse_args()
    if platform.system() != 'Darwin' or platform.machine() != 'arm64':
        raise SystemExit('Build this tested Apple Silicon candidate on an Apple Silicon Mac.')
    stage = args.output.resolve()
    if stage.exists():
        raise SystemExit('Refusing to replace a candidate. Choose a new --output directory.')
    app = stage / 'Jornia.app'
    contents = app / 'Contents'
    resources = contents / 'Resources'
    (contents / 'MacOS').mkdir(parents=True)
    (resources / 'bin').mkdir(parents=True)
    (resources / 'macos').mkdir()
    for name in ('supervisor.py', 'install_addon.py'):
        shutil.copy2(ROOT / 'apps/macos' / name, resources / 'macos' / name)
    for directory, names in {
        'companion': ('reader.py', 'protocol.py', 'invite.py'),
        'experiments/return_bus': ('bus.py', 'block.py'),
        'experiments/ocr': ('bridge.py',),
    }.items():
        target = resources / 'scripts' / directory
        target.mkdir(parents=True)
        for name in names:
            shutil.copy2(ROOT / 'scripts' / directory / name, target / name)
    env = os.environ | {'GOCACHE': str(ROOT / '.cache/go-build'), 'GOMODCACHE': str(ROOT / '.cache/go-mod'), 'CGO_ENABLED': '0', 'GOOS': 'darwin', 'GOARCH': 'arm64'}
    run(['go', 'build', '-trimpath', '-o', resources / 'bin/jornia', './apps/companion'], cwd=ROOT, env=env)
    cache = ROOT / '.cache/objc-module-cache'
    cache.mkdir(parents=True, exist_ok=True)
    common = ['clang', '-fobjc-arc', '-fmodules', '-fmodules-cache-path=' + str(cache), '-mmacosx-version-min=14.0']
    run(common + ['-framework', 'Cocoa', ROOT / 'apps/macos/Jornia.m', '-o', contents / 'MacOS/Jornia'])
    frameworks = ['Foundation', 'AppKit', 'Vision', 'CoreGraphics', 'ScreenCaptureKit', 'CoreImage', 'CoreMedia']
    run(common + [arg for framework in frameworks for arg in ('-framework', framework)] + [ROOT / 'scripts/experiments/ocr/capture.m', '-o', resources / 'bin/jornia-capture'])
    version = bundle(resources / 'Runtime')
    # A helper's deployment target is not sufficient: the bundled runtime may
    # require a newer OS. Declare the highest target of every linked native file.
    minimum = (14, 0)
    for path in app.rglob('*'):
        if native(path):
            output = subprocess.check_output(['otool', '-l', str(path)], text=True)
            for value in re.findall(r'\bminos ([\d.]+)', output):
                minimum = max(minimum, tuple(int(part) for part in value.split('.')))
    minimum_os = '.'.join(str(part) for part in minimum)
    ca_file = ssl.get_default_verify_paths().cafile
    if not ca_file or not Path(ca_file).is_file():
        raise SystemExit('A public CA certificate bundle is required for packaged HTTPS.')
    shutil.copy2(ca_file, resources / 'cacert.pem')
    prepare_addons(resources / 'AddOns')
    with (contents / 'Info.plist').open('wb') as target:
        plistlib.dump(dict(CFBundleIdentifier='com.aheige.jornia', CFBundleName='Jornia', CFBundleDisplayName='Jornia', CFBundleExecutable='Jornia', CFBundlePackageType='APPL', CFBundleShortVersionString='0.9.0', CFBundleVersion='9', LSMinimumSystemVersion=minimum_os, NSHighResolutionCapable=True, NSScreenCaptureUsageDescription='Jornia reads the small in-game chat code locally. Screenshots are never uploaded.'), target)
    notices = stage / 'THIRD-PARTY'
    notices.mkdir()
    shutil.copy2(Path(sys.base_prefix) / 'lib' / ('python' + version) / 'LICENSE.txt', notices / 'Python-LICENSE.txt')
    for path in [Path('/opt/homebrew/opt/openssl@3/share/doc/openssl/LICENSE.txt'), Path('/opt/homebrew/opt/xz/share/doc/xz/COPYING'), Path('/opt/homebrew/opt/bzip2/LICENSE'), Path('/opt/homebrew/opt/mpdecimal/share/doc/mpdecimal/LICENSE.txt'), Path('/opt/homebrew/opt/ca-certificates/share/ca-certificates/README')]:
        if path.is_file():
            shutil.copy2(path, notices / (path.parts[4] + '-' + path.name))
    raw = subprocess.check_output(['go', 'list', '-deps', '-f', '{{if .Module}}{{.Module.Dir}}{{end}}', './apps/companion'], cwd=ROOT, env=env, text=True)
    for directory in sorted(set(raw.splitlines())):
        if directory and Path(directory) != ROOT:
            for pattern in ('LICENSE*', 'NOTICE*', 'COPYING*'):
                for path in Path(directory).glob(pattern):
                    if path.is_file() and not path.name.startswith('._'):
                        shutil.copy2(path, notices / (Path(directory).name + '-' + path.name))
    goroot = Path(subprocess.check_output(['go', 'env', 'GOROOT'], env=env, text=True).strip())
    shutil.copy2(next(path for path in [goroot / 'LICENSE', goroot.parent / 'LICENSE'] if path.is_file()), notices / 'Go-LICENSE')
    for package in ('tailwindcss', 'daisyui'):
        paths = list((ROOT / 'apps/companion/web/node_modules' / package).glob('LICENSE*'))
        if not paths:
            raise SystemExit('Missing CSS licence: ' + package)
        for path in paths:
            shutil.copy2(path, notices / (package + '-' + path.name))
    shutil.copy2(ROOT / 'apps/addon/Jornia/LICENSE.txt', resources / 'AddOns/Wisp/LICENSE.txt')
    # Sign locally only to permit execution after binary relocation. This does not create a trusted release.
    # ExFAT stores builder metadata in AppleDouble siblings; they are not part of our product.
    for path in stage.rglob('._*'):
        if path.is_file():
            path.unlink()
    # The external workspace may be ExFAT. codesign creates AppleDouble sidecars there
    # which invalidate its own recursive seal. Seal a temporary APFS copy, then return it.
    with tempfile.TemporaryDirectory(prefix='jornia-seal-', dir='/private/tmp') as temporary:
        sealed = Path(temporary) / 'Jornia.app'
        shutil.copytree(app, sealed, copy_function=shutil.copy, ignore=shutil.ignore_patterns('._*'))
        for target in signing_targets(sealed):
            if target.is_file():
                target.chmod(0o755)
            run(['codesign', '--force', '--sign', '-', target], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        run(['codesign', '--verify', '--deep', '--strict', sealed])
        shutil.rmtree(app)
        shutil.copytree(sealed, app, copy_function=shutil.copy)
    for path in stage.rglob('._*'):
        if path.is_file():
            path.unlink()
    run(['codesign', '--verify', '--deep', '--strict', app])
    clean_env = {'HOME': str(Path.home()), 'PATH': '/usr/bin:/bin', 'PYTHONHOME': str(resources / 'Runtime'), 'PYTHONNOUSERSITE': '1', 'PYTHONDONTWRITEBYTECODE': '1', 'SSL_CERT_FILE': str(resources / 'cacert.pem')}
    run([resources / 'Runtime/bin/python3', '-s', '-S', '-c', "import ssl,urllib.request,fcntl,sqlite3,hashlib,ctypes,zlib; print('Private runtime imports passed')"], env=clean_env)
    # No native dependency may require the developer's Homebrew or checkout after relocation.
    for path in app.rglob('*'):
        if native(path):
            for dep in dependencies(path):
                if not dep.startswith(('@', '/usr/lib/', '/System/Library/')):
                    raise SystemExit('Non-relocatable binary dependency: ' + str(path.relative_to(app)) + ': ' + dep)
    (stage / 'MAC-INSTALL.md').write_text('Jornia 0.9.0 native Mac candidate — Apple Silicon, macOS '+minimum_os+'+\n\nLocal development candidate, ad-hoc signed only. NOT Developer ID signed or notarized. Do not disable Gatekeeper. Do not distribute this as a trusted download.\n\nOpen Jornia.app, choose your Forever folder, install the addon before your next normal game launch, allow local screen capture and start in-game chat. If the addon is already installed, installation is optional. Quit older Jornia desktop/readers before starting. Open /jornia in WoW; enter an invite in Settings if needed. Keep the chat visible until the answer arrives. Images stay local; submitted text and approved context go to AI.\n\nStop or quit Jornia to restore owned connection files. Your saved character journals are preserved. Automatic app updates and background chat delivery are not supported in this candidate.\n')
    manifest = {str(path.relative_to(stage)): hashlib.sha256(path.read_bytes()).hexdigest() for path in stage.rglob('*') if path.is_file() and not path.name.startswith('._')}
    (stage / 'SHA256SUMS.json').write_text(json.dumps(manifest, indent=2) + '\n')
    print('Native Mac candidate (not notarized):', app)


if __name__ == '__main__':
    main()
