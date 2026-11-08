import json,struct,time,unittest
from protocol import parse,reply,frame,checksum
class ProtocolTest(unittest.TestCase):
 def test_unicode_identity_damage_and_frame(self):
  ident='123456781794000000000001';guid='Player-test';binding=f'{checksum(guid):05d}';nonce='12345678'
  raw=json.dumps(dict(q='Var nu? 中文',s=None),ensure_ascii=False,separators=(',',':'))
  wire=f'WISPBEGIN3 {ident} {binding} {nonce} {checksum(ident+binding+nonce+raw):05d}\n{raw.encode().hex()}\nWISPEND'
  p=parse(wire.splitlines(),{guid:'local-one'});self.assertEqual(p['question'],'Var nu? 中文')
  self.assertIsNone(parse(wire.replace('BEGIN3','BEGIN2').splitlines(),{guid:'local-one'}))
  self.assertIsNone(parse(wire.splitlines(),{'Player-other':'local-two'}))
  b=reply(p,dict(text='Svar åäö',sources=[dict(id='foreverdb:build:test')]))
  self.assertEqual(b[2:26],ident.encode());n=struct.unpack('>H',b[26:28])[0];self.assertEqual(b[28:28+n].decode(),'Svar åäö');self.assertEqual(b[28+n],1)
  self.assertEqual(len(frame(b,nonce,binding))%8,0)
 def test_oversize_is_controlled_error(self):
  p=dict(id='0'*24);b=reply(p,dict(text='x'*2000));self.assertEqual(b[1],1)
if __name__=='__main__':unittest.main()

class RichSnapshotTest(unittest.TestCase):
 def test_current_progress_equipment_and_skills(self):
  from protocol import expand_snapshot
  v=[int(time.time()),16,'Deathknell','70790',0,[[376,'The Damned',0,0,[[1,4,6,0]]]],dict(c=8,r=5,e=[[16,35]],p=[[129,12,75]],m=[])]
  s=expand_snapshot(v);self.assertEqual(s['level'],16);self.assertEqual(s['quests'][0]['objectives'][0]['fulfilled'],4);self.assertEqual(s['equipment'][0]['item_id'],35);self.assertEqual(s['professions'][0]['rank'],12)
  v[6]['e'].append([16,36])
  with self.assertRaises(ValueError):expand_snapshot(v)

class InviteTest(unittest.TestCase):
 def test_one_use_masks_restore_and_no_plaintext_wire(self):
  import tempfile,pathlib
  from invite import InvitePads
  with tempfile.TemporaryDirectory() as folder:
   root=pathlib.Path(folder)
   for n in range(67,81):
    p=root/'Interface/AddOns'/f'WispOCRInbox{n:03d}';p.mkdir(parents=True);(p/'Inbox.lua').write_text('-- original\n');(p/(p.name+'_Camelot.toc')).write_text('## Dependencies: WispOCR\n')
   pads=InvitePads(root);ident,pad=next(iter(pads.keys.items()));token=bytes.fromhex('ab'*32);cipher=bytes(a^b for a,b in zip(token,pad)).hex();packet=dict(activation=dict(pad=ident,cipher=cipher))
   self.assertNotIn(token.hex(),json.dumps(packet));self.assertEqual(pads.decrypt(packet),token.hex())
   with self.assertRaises(ValueError):pads.decrypt(packet)
   pads.restore()
   self.assertEqual((root/'Interface/AddOns/WispOCRInbox067/Inbox.lua').read_text(),'-- original\n');self.assertFalse(list(root.rglob('*.wispconnect-hold')))

class FirstQuestionIdentityTest(unittest.TestCase):
 def test_unsaved_character_can_register_but_forged_binding_is_rejected(self):
  guid='Player-new';ident='123456781794000000000001';binding=f'{checksum(guid):05d}';nonce='12345678'
  raw=json.dumps(dict(q='What now?',s=None,c=['installation-1',guid,'New Hero','Realm']),separators=(',',':'))
  wire=f'WISPBEGIN3 {ident} {binding} {nonce} {checksum(ident+binding+nonce+raw):05d}\n{raw.encode().hex()}\nWISPEND'
  packet=parse(wire.splitlines(),{})
  self.assertEqual(packet['registration']['character']['id'],guid)
  self.assertIsNone(packet['character_id'])
  self.assertIsNone(parse(wire.replace(binding,'00000',1).splitlines(),{}))
