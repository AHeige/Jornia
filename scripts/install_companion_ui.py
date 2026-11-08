#!/usr/bin/env python3
"""Install the local Lua companion view without touching journal data or restarting WoW."""
import argparse,datetime,pathlib,shutil
p=argparse.ArgumentParser(description=__doc__);p.add_argument('--root',type=pathlib.Path,required=True);a=p.parse_args()
repo=pathlib.Path(__file__).resolve().parents[1]
addon=a.root/'Interface/AddOns/Wisp';toc=addon/'Wisp_Camelot.toc'
if not toc.is_file() or not (a.root/'WTF').is_dir():raise SystemExit('Expected installed Jornia addon in the Forever client root')
backup=repo/'.cache'/('addon-ui-backup-'+datetime.datetime.now().strftime('%Y%m%d-%H%M%S'))
shutil.copytree(addon,backup,ignore=shutil.ignore_patterns('._*','.DS_Store'))
for name in ['Wisp.lua','State.lua','Companion.lua','Bindings.lua','Navigation.lua','Transport.lua','Media/expedition.tga','Media/avatar.tga']:
 source=repo/'apps/addon/Jornia'/name
 if not source.is_file():raise SystemExit('Missing prepared source/artwork: '+name)
for name in ['Wisp.lua','State.lua','Companion.lua','Bindings.lua','Navigation.lua','Transport.lua','Media/expedition.tga','Media/avatar.tga']:
 target=addon/name;target.parent.mkdir(parents=True,exist_ok=True);temp=target.with_suffix(target.suffix+'.wisp-tmp')
 shutil.copyfile(repo/'apps/addon/Jornia'/name,temp);temp.chmod(0o644);temp.replace(target)
lines=toc.read_text().splitlines()
lines=['## Version: 0.8.0-local' if line.startswith('## Version:') else '## Title: Jornia' if line.startswith('## Title:') else line for line in lines]
for name in ['State.lua','Companion.lua','Bindings.lua','Navigation.lua','Transport.lua']:
 if name not in lines:lines.append(name)
lines=[line for line in lines if line!='State.lua'];lines.insert(lines.index('Wisp.lua')+1,'State.lua')
temp=toc.with_suffix('.toc.wisp-tmp');temp.write_text('\n'.join(lines)+'\n');temp.chmod(0o644);temp.replace(toc)
print('Installed local state, journal and companion view. Existing WTF/player data preserved.')
print('Backup:',backup)
print('No game restart/reload performed. An unused installed LoD slot can activate this during an existing session.')
