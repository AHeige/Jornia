#!/usr/bin/env python3
"""Build self-contained Windows/macOS test bundles. Run from repository root."""
import hashlib
import argparse
import os
import plistlib
from pathlib import Path
import shutil
import subprocess
import zipfile

root = Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser();parser.add_argument('--use-built-web',action='store_true',help='Reuse unchanged checked-in web CSS without reinstalling dependencies');args=parser.parse_args()
version = '0.7.0'
for system,arch in [('windows','amd64'),('darwin','arm64')]:
    if (root/'dist'/f'wisp-{version}-{system}-{arch}.zip').exists():
        raise SystemExit('Release already exists; bump the version before rebuilding published artifacts.')
env = os.environ | {'GOCACHE':str(root/'.cache/go-build'),'GOMODCACHE':str(root/'.cache/go-mod'),'CGO_ENABLED':'0'}
if not args.use_built_web:
    subprocess.run(['npm','ci','--prefix','apps/companion/web','--no-audit','--no-fund'],cwd=root,check=True)
    subprocess.run(['npm','run','build','--prefix','apps/companion/web'],cwd=root,check=True)
for system,arch in [('windows','amd64'),('darwin','arm64')]:
    name=f'wisp-{version}-{system}-{arch}'
    stage=root/'dist'/name
    stage.mkdir(parents=True,exist_ok=True)
    for stale in ['Prepare-Wisp-Connection.ps1','Start-VM-Wisp.ps1']:
        (stage/stale).unlink(missing_ok=True)
    binary=stage/('wisp.exe' if system=='windows' else 'wisp')
    subprocess.run(['go','build','-trimpath','-o',str(binary),'./apps/companion'],cwd=root,env=env|{'GOOS':system,'GOARCH':arch},check=True)
    addon=stage/'AddOns/Wisp'
    shutil.copytree(root/'apps/addon/Wisp',addon,dirs_exist_ok=True,ignore=shutil.ignore_patterns('._*','.DS_Store'))
    shutil.copy2(root/'docs/wow-first-test.md',stage/'START-HERE.md')
    shutil.copy2(root/'docs/automatic-journal.md',stage/'automatic-journal.md')
    shutil.copy2(root/'docs/companion-dependencies.md',stage/'DEPENDENCIES.md')
    shutil.copy2(root/f'docs/release-{version}.md',stage/'RELEASE-NOTES.md')
    if system=='windows':
        (stage/'Start-Wisp.cmd').write_bytes(b'@echo off\r\ncd /d "%~dp0"\r\nwisp.exe %*\r\nif errorlevel 1 pause\r\n')
        shutil.copy2(root/'scripts/install-addon.ps1',stage/'Install-Addon.ps1')
        (stage/'Start-Wisp-VM.cmd').write_bytes(b'@echo off\r\ncd /d "%~dp0"\r\nwisp.exe -vm %*\r\nif errorlevel 1 pause\r\n')
    if system=='darwin':
        app=stage/'Wisp.app/Contents'
        (app/'MacOS').mkdir(parents=True,exist_ok=True)
        shutil.move(binary,app/'MacOS/Wisp')
        with open(app/'Info.plist','wb') as info:
            plistlib.dump({'CFBundleIdentifier':'com.aheige.wisp','CFBundleName':'Wisp','CFBundleDisplayName':'Wisp','CFBundleExecutable':'Wisp','CFBundlePackageType':'APPL','CFBundleShortVersionString':version,'CFBundleVersion':version,'LSMinimumSystemVersion':'12.0','LSUIElement':True,'NSHighResolutionCapable':True},info)
        (stage/'MAC-INSTALL.md').write_text('Wisp '+version+' — Apple Silicon\n\nThis candidate is not Developer ID signed or notarized. Do not disable Gatekeeper. A separately labelled notarized build will be published after Apple accepts it. The app opens your normal browser and stores data in ~/Library/Application Support/Wisp. Addon files are in AddOns/Wisp.\n')
    notices=stage/'THIRD-PARTY';notices.mkdir(exist_ok=True)
    # Include licence files for modules actually linked into this companion, not developer credentials or runtime state.
    raw=subprocess.check_output(['go','list','-deps','-f','{{if .Module}}{{.Module.Dir}}{{end}}','./apps/companion'],cwd=root,env=env,text=True)
    for directory in sorted(set(raw.splitlines())):
        if not directory or Path(directory)==root: continue
        source=Path(directory)
        for pattern in ['LICENSE*','COPYING*','COPYRIGHT*','NOTICE*']:
            for file in source.glob(pattern):
                if file.is_file() and not file.name.startswith('._'):
                    shutil.copy2(file,notices/(source.name+'-'+file.name))
    goroot=subprocess.check_output(['go','env','GOROOT'],cwd=root,env=env,text=True).strip()
    go_license=Path(goroot)/'LICENSE'
    if not go_license.exists():go_license=Path(goroot).parent/'LICENSE'
    shutil.copy2(go_license,notices/'Go-LICENSE')
    for package in ['tailwindcss','daisyui']:
        directory=root/'apps/companion/web/node_modules'/package
        candidates=list(directory.glob('LICENSE*'))
        if not candidates and args.use_built_web:
            candidates=list((root/'dist/wisp-0.6.0-darwin-arm64/THIRD-PARTY').glob(package+'-LICENSE*'))
        if not candidates:raise SystemExit('Missing licence for '+package)
        for file in candidates:
            if file.is_file():shutil.copy2(file,notices/(package+'-'+file.name))
    archive=root/'dist'/f'{name}.zip'
    with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED) as z:
        for file in stage.rglob('*'):
            if file.is_file() and not file.name.startswith('._'):
                z.write(file,file.relative_to(stage.parent))
    digest=hashlib.sha256(archive.read_bytes()).hexdigest()
    archive.with_suffix('.zip.sha256').write_text(digest+'  '+archive.name+'\n')
    print(archive,archive.stat().st_size,'bytes',digest)
