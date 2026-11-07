"""One-use local masks for invite entry. Codes never appear in OCR as plaintext.
Each 32-byte invite is XORed with a fresh, equally long CSPRNG pad, used once.
No reusable home-grown cipher; unused installed LoD slots carry the local pads.
"""
import hashlib,pathlib,re,secrets
class InvitePads:
 def __init__(self,root):
  self.keys={};self.original={};self.written={}
  try:
   for n in range(67,81):
    folder=root/'Interface/AddOns'/f'WispOCRInbox{n:03d}';p=folder/'Inbox.lua';toc=folder/(folder.name+'_Camelot.toc');held=p.with_name('Inbox.lua.wispconnect-hold')
    if folder.is_symlink() or p.is_symlink() or toc.is_symlink() or held.exists() or not p.is_file() or b'## Dependencies: WispOCR' not in toc.read_bytes():raise ValueError('Invite slot needs recovery')
    raw=p.read_bytes();ident=secrets.token_hex(8);pad=secrets.token_bytes(32)
    with held.open('xb')as f:f.write(raw)
    held.chmod(0o600);self.original[p]=held
    text=f'WispConnectPads=WispConnectPads or {{}};WispConnectPads[{n}]={{id="{ident}",hex="{pad.hex()}"}}\n'
    temp=p.with_suffix('.connect-tmp');temp.write_text(text);temp.chmod(0o600);temp.replace(p)
    self.written[p]=hashlib.sha256(p.read_bytes()).hexdigest();self.keys[ident]=pad
  except Exception:self.restore();raise
 def decrypt(self,packet):
  value=packet['activation'];ident=value['pad'];cipher=bytes.fromhex(value['cipher']);pad=self.keys.pop(ident,None)
  if pad is None or len(cipher)!=32:raise ValueError('Invite mask expired or already used')
  return bytes(a^b for a,b in zip(cipher,pad)).hex()
 def restore(self):
  self.keys.clear()
  for p,held in self.original.items():
   if p.is_symlink() or held.is_symlink():raise ValueError('Invite recovery symlink rejected')
   if p in self.written and hashlib.sha256(p.read_bytes()).hexdigest()!=self.written[p]:raise ValueError('Invite file changed; originals preserved')
   held.replace(p)
  self.original.clear()
