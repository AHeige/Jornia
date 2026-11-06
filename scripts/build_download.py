#!/usr/bin/env python3
"""Build the static download page; optionally export its public files for Pages."""
from pathlib import Path
import argparse
import re
import shutil
import subprocess

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument('--pages-output', type=Path)
parser.add_argument('--release-repo', default='AHeige/wisp-site')
args = parser.parse_args()
compiler = root / 'apps/companion/web/node_modules/.bin/tailwindcss'
if not compiler.exists():
    raise SystemExit('Install existing UI dependencies: npm ci --prefix apps/companion/web')
subprocess.run([str(compiler), '-i', 'apps/download/input.css', '-o', 'apps/download/download.css', '--minify'], cwd=root, check=True)
if args.pages_output:
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', args.release_repo):
        raise SystemExit('Invalid release repository')
    source = root / 'apps/download'
    html = (source / 'index.html').read_text()
    images = set(re.findall(r'/media/([a-zA-Z0-9_.-]+)', html))
    def release(match):
        name, version = match.group(1), match.group(2)
        return f'https://github.com/{args.release_repo}/releases/download/v{version}/{name}'
    html = re.sub(r'/releases/(wisp-([0-9]+\.[0-9]+\.[0-9]+)-(?:windows-amd64|darwin-arm64)\.zip(?:\.sha256)?)', release, html)
    html = html.replace('/download.css', './download.css').replace('/media/', './media/')
    output = args.pages_output.resolve()
    if output == root or output == source:
        raise SystemExit('Pages output must be separate from source')
    (output / 'media').mkdir(parents=True, exist_ok=True)
    (output / 'index.html').write_text(html)
    shutil.copy2(source / 'download.css', output / 'download.css')
    for name in images:
        shutil.copy2(source / 'media' / name, output / 'media' / name)
    (output / '.nojekyll').touch()
    print(f'Pages export: {output} ({len(images)} public images)')
