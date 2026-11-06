#!/usr/bin/env python3
"""Build self-contained Windows/macOS test bundles. Run from repository root."""
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import zipfile

root = Path(__file__).resolve().parents[1]
version = '0.5.2'
for system,arch in [('windows','amd64'),('darwin','arm64')]:
    if (root/'dist'/f'wisp-{version}-{system}-{arch}.zip').exists():
        raise SystemExit('Release already exists; bump the version before rebuilding published artifacts.')
env = os.environ | {'GOCACHE':str(root/'.cache/go-build'),'GOMODCACHE':str(root/'.cache/go-mod'),'CGO_ENABLED':'0'}
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
    if system=='windows':
        (stage/'Start-Wisp.cmd').write_bytes(b'@echo off\r\ncd /d "%~dp0"\r\nwisp.exe %*\r\nif errorlevel 1 pause\r\n')
        shutil.copy2(root/'scripts/install-addon.ps1',stage/'Install-Addon.ps1')
        (stage/'Start-Wisp-VM.cmd').write_bytes(b'@echo off\r\ncd /d "%~dp0"\r\nwisp.exe -vm %*\r\nif errorlevel 1 pause\r\n')
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
        for file in directory.glob('LICENSE*'):
            if file.is_file():shutil.copy2(file,notices/(package+'-'+file.name))
    archive=root/'dist'/f'{name}.zip'
    with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED) as z:
        for file in stage.rglob('*'):
            if file.is_file() and not file.name.startswith('._'):
                z.write(file,file.relative_to(stage.parent))
    digest=hashlib.sha256(archive.read_bytes()).hexdigest()
    archive.with_suffix('.zip.sha256').write_text(digest+'  '+archive.name+'\n')
    print(archive,archive.stat().st_size,'bytes',digest)
