#!/usr/bin/env python3
"""Build only the public download page stylesheet; never rebuild a published client."""
from pathlib import Path
import subprocess
root=Path(__file__).resolve().parents[1]
compiler=root/'apps/companion/web/node_modules/.bin/tailwindcss'
if not compiler.exists():
    raise SystemExit('Install existing UI dependencies: npm ci --prefix apps/companion/web')
subprocess.run([str(compiler),'-i','apps/download/input.css','-o','apps/download/download.css','--minify'],cwd=root,check=True)
