#!/usr/bin/env python3
"""Sign/notarize a prepared Jornia.app. Uses installed Developer ID and Keychain profile only."""
import argparse
import hashlib
import json
import pathlib
import subprocess
import tempfile
from macos_bundle import signing_targets

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('app', type=pathlib.Path)
p.add_argument('--identity', required=True, help='Installed Developer ID Application identity SHA-1')
p.add_argument('--notary-profile', required=True, help='Previously stored notarytool Keychain profile')
p.add_argument('--output', type=pathlib.Path, required=True)
a = p.parse_args()
app = a.app.resolve()
if app.name != 'Jornia.app' or not (app/'Contents/MacOS/Jornia').is_file() or not (app.parent/'AddOns/Wisp').is_dir() or not (app.parent/'START-HERE.md').is_file():
    raise SystemExit('Expected prepared Jornia.app')
if app.parent in a.output.resolve().parents:
    raise SystemExit('Output ZIP must be outside the candidate staging directory')
if a.output.exists():
    raise SystemExit('Refusing to overwrite release artifact')
identities = subprocess.check_output(['security','find-identity','-v','-p','codesigning'],text=True)
if not any(a.identity in line and 'Developer ID Application:' in line for line in identities.splitlines()):
    raise SystemExit('An installed valid Developer ID Application identity is required')
# Seal capture/runtime helpers and other nested code before the outer app.
# No --deep signing or relaxed entitlements. Bundle inspection fails on escaping links.
for target in signing_targets(app):
    subprocess.run(['codesign','--force','--options','runtime','--timestamp','--sign',a.identity,str(target)],check=True)
subprocess.run(['codesign','--verify','--deep','--strict','--verbose=2',str(app)],check=True)
with tempfile.TemporaryDirectory(prefix='wisp-notary-',dir=app.parent) as directory:
    upload=pathlib.Path(directory)/'Jornia.zip'
    subprocess.run(['ditto','-c','-k','--keepParent',str(app),str(upload)],check=True)
    # Submission can take minutes. Keep this interactive command visible to the operator.
    result=json.loads(subprocess.check_output(['xcrun','notarytool','submit',str(upload),'--keychain-profile',a.notary_profile,'--wait','--output-format','json'],text=True))
    if result.get('status')!='Accepted':
        raise SystemExit('Apple did not accept notarization. No release ZIP created. Submission: '+str(result.get('id')))
subprocess.run(['xcrun','stapler','staple',str(app)],check=True)
subprocess.run(['xcrun','stapler','validate',str(app)],check=True)
subprocess.run(['spctl','--assess','--type','execute','--verbose=2',str(app)],check=True)
a.output.parent.mkdir(parents=True,exist_ok=True)
# Include the addon, instructions and dependency licences as well as the stapled app.
(app.parent/'MAC-INSTALL.md').write_text('Jornia — Apple Silicon\n\nThis app passed Developer ID signing, Apple notarization, stapling and Gatekeeper assessment during packaging. Move Jornia.app to Applications and open it. It opens your normal browser. Follow START-HERE.md to install AddOns/Wisp and connect your game folder. Keep OS security protections enabled. Your private data stays in ~/Library/Application Support/Wisp.\n')
subprocess.run(['ditto','-c','-k','--keepParent',str(app.parent),str(a.output)],check=True)
digest=hashlib.sha256(a.output.read_bytes()).hexdigest()
a.output.with_suffix(a.output.suffix+'.sha256').write_text(digest+'  '+a.output.name+'\n')
print('Signed, notarized, stapled and Gatekeeper-assessed:',a.output)
