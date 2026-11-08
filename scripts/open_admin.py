#!/usr/bin/env python3
"""Open Jornia's private invite admin through an owner SSH connection."""
import argparse,ipaddress,pathlib,socket,subprocess,time,urllib.error,urllib.request,webbrowser
p=argparse.ArgumentParser(description=__doc__);p.add_argument('--no-browser',action='store_true');a=p.parse_args()
ssh=['ssh','-o','BatchMode=yes','-i',str(pathlib.Path.home()/'.ssh/ubuntu_dev'),'-p','2222']
command='docker inspect -f \'{{(index .NetworkSettings.Networks "wisp-private_backend").IPAddress}}\' wisp-private-admin-1'
ip=subprocess.check_output(ssh+['leon@178.174.244.160',command],text=True).strip();address=ipaddress.ip_address(ip)
if address.version!=4 or not address.is_private:raise SystemExit('Expected the private Jornia admin container address')
with socket.socket()as probe:
 if probe.connect_ex(('127.0.0.1',18983))==0:raise SystemExit('Local port 18983 is already in use. Existing listener preserved.')
tunnel=subprocess.Popen(ssh+['-o','ExitOnForwardFailure=yes','-N','-L',f'127.0.0.1:18983:{ip}:8091','leon@178.174.244.160'])
try:
 for _ in range(30):
  if tunnel.poll()is not None:raise SystemExit('SSH tunnel could not start')
  try:urllib.request.urlopen('http://127.0.0.1:18983/',timeout=1)
  except urllib.error.HTTPError as e:
   if e.code==401:break
  except OSError:pass
  time.sleep(.1)
 else:raise SystemExit('Private admin did not become reachable')
 print('Private admin: http://127.0.0.1:18983/\nUser: admin\nPassword: read your owner-only .cache/admin-access.txt\nKeep this terminal open; Ctrl+C closes the tunnel. No public admin port is used.',flush=True)
 if not a.no_browser:webbrowser.open('http://127.0.0.1:18983/')
 tunnel.wait()
except KeyboardInterrupt:pass
finally:tunnel.terminate();tunnel.wait(timeout=5)
