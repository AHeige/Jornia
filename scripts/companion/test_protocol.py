import json,struct,time,unittest,zlib
from protocol import parse,reply,frame,checksum
class ProtocolTest(unittest.TestCase):
 def test_unicode_identity_damage_and_frame(self):
  ident='123456781794000000000001';guid='Player-test';binding=f'{checksum(guid):05d}';nonce='12345678'
  raw=json.dumps(dict(q='Var nu? 中文',s=None),ensure_ascii=False,separators=(',',':'))
  wire=f'WISPBEGIN3 {ident} {binding} {nonce} {checksum(ident+binding+nonce+raw):05d}\n{raw.encode().hex()}\nWISPEND'
  p=parse(wire.splitlines(),{guid:'local-one'});self.assertEqual(p['question'],'Var nu? 中文')
  self.assertEqual(parse(wire.replace('c','с').splitlines(),{guid:'local-one'})['question'],p['question'])
  self.assertIsNone(parse(wire.replace('c','d',1).splitlines(),{guid:'local-one'}))
  self.assertIsNone(parse(wire.replace('BEGIN3','BEGIN2').splitlines(),{guid:'local-one'}))
  self.assertIsNone(parse(wire.splitlines(),{'Player-other':'local-two'}))
  b=reply(p,dict(text='Svar åäö',sources=[dict(id='foreverdb:build:test')]))
  self.assertEqual(b[2:26],ident.encode());n=struct.unpack('>H',b[26:28])[0];self.assertEqual(b[28:28+n].decode(),'Svar åäö');self.assertEqual(b[28+n],1)
  self.assertEqual(len(frame(b,nonce,binding))%8,0)
 def test_oversize_is_controlled_error(self):
  p=dict(id='0'*24);b=reply(p,dict(text='x'*2000));self.assertEqual(b[1],1)

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

class QuestTextTest(unittest.TestCase):
 def test_bounded_directions_and_legacy_quests(self):
  from protocol import expand_snapshot
  row=[376,'The Damned',0,0,[],['Go west.','Collect paws.','At the farm.',0]]
  v=[int(time.time()),16,'Deathknell','70790',0,[row]]
  q=expand_snapshot(v)['quests'][0];self.assertEqual(q['description'],'Go west.');self.assertFalse(q['text_truncated'])
  row[5][0]='å'*513
  with self.assertRaises(ValueError):expand_snapshot(v)
  row.pop();self.assertNotIn('description',expand_snapshot(v)['quests'][0])

class OCRAlternativesTest(unittest.TestCase):
 def test_alternatives_require_original_checksum_and_preserve_bounds(self):
  from protocol import parse_observation
  guid='Player-test';ident='123456781794000000000001';binding=f'{checksum(guid):05d}';nonce='12345678'
  raw=json.dumps(dict(q='Where next?',s=None),separators=(',',':'));encoded=raw.encode().hex()
  good=[f'WISPBEGIN4 {ident} {binding} {nonce} {zlib.adler32((ident+binding+nonce+raw).encode()):010d}',encoded[:96],encoded[96:],'WISPEND']
  if not good[2]:good.pop(2)
  rows=[[x] for x in good];rows[1]=['f'+good[1][1:],good[1]]
  packet=parse_observation(dict(lines=[x[0] for x in rows],candidates=rows),{guid:'local'})
  self.assertEqual(packet['question'],'Where next?')
  rows[1]=['f'+good[1][1:]];self.assertIsNone(parse_observation(dict(lines=[x[0] for x in rows],candidates=rows),{guid:'local'}))

class NumericOCRTest(unittest.TestCase):
 def test_header_spaces_and_digit_glyphs_still_need_full_checksum(self):
  from protocol import normalize_ocr_lines,parse
  guid='Player-test';ident='123456781794000000000001';binding=f'{checksum(guid):05d}';nonce='12345678'
  raw=json.dumps(dict(q='Where next?',s=None),separators=(',',':'));digest=f'{zlib.adler32((ident+binding+nonce+raw).encode()):010d}'
  header=f'WISPBEGIN4 {ident}{binding[:4]} {binding[4:]} {nonce} {digest}'
  lines=[header,raw.encode().hex(),'WISPEND'];self.assertEqual(parse(lines,{guid:'local'})['question'],'Where next?')
  self.assertIsNone(parse([header,raw.encode().hex().replace('7b','7c',1),'WISPEND'],{guid:'local'}))
  self.assertEqual(normalize_ocr_lines(['WBACK 123456781271$0010008000000000000000'])[0],'WBACK 12345678 12715 001 0008 0000000000 00000')

class QRTest(unittest.TestCase):
 def test_exact_unicode_context_receipts_and_rejected_damage(self):
  from protocol import qr_lines,parse_observation
  guid='Player-test';ident='123456781794000000000001';binding=f'{checksum(guid):05d}';nonce='12345678'
  state=[int(time.time()),2,'Deathknell','70790',0,[[376,'The Damned',0,0,[[1,4,6,0,'Scavenger Paw']],['Find wolves south.','Collect paws.','',0]]]]
  raw=json.dumps(dict(q='Vad saknas? 中文',s=state),ensure_ascii=False,separators=(',',':'))
  header=f'{ident} {binding} {nonce} {zlib.adler32((ident+binding+nonce+raw).encode()):010d}'
  value='JORNIAQR1\n'+header+'\n'+raw
  packet=parse_observation(dict(qr=[value]),{guid:'local'})
  self.assertEqual(packet['question'],'Vad saknas? 中文');self.assertEqual(packet['snapshot']['quests'][0]['objectives'][0]['fulfilled'],4)
  self.assertIsNone(parse_observation(dict(qr=[value.replace('south','north')]),{guid:'local'}))
  self.assertIsNone(parse_observation(dict(qr=[value,value]),{guid:'local'}))
  self.assertIsNone(parse_observation(dict(qr=[value]),{'Player-wrong':'other'}))
  self.assertIsNone(qr_lines(dict(qr=['https://example.com'])))
  ack='WBACK 12345678 12715 001 0008 0000000000 00000'
  self.assertEqual(qr_lines(dict(qr=[ack])),[ack])
  self.assertIsNone(qr_lines(dict(qr=[ack.replace('12715','1271S')])))

class CompletedQuestTest(unittest.TestCase):
 def test_turn_in_evidence_and_active_conflict(self):
  from protocol import expand_snapshot
  now=int(time.time());meta=dict(c=8,r=5,e=[],p=[],m=[],d=[[376,'The Damned',now,'turn_in_event',['Scavenger Paw']]])
  state=[now,3,'Deathknell','70790',0,[],meta]
  self.assertEqual(expand_snapshot(state)['completed_quests'][0]['id'],376)
  state[5]=[[376,'The Damned',1,0]]
  with self.assertRaises(ValueError):expand_snapshot(state)
  state[5]=[];meta['d'][0][3]='missing_from_log'
  with self.assertRaises(ValueError):expand_snapshot(state)

class PersonalContextTest(unittest.TestCase):
 def test_observed_identity_and_partial_journey(self):
  from protocol import expand_snapshot
  now=int(time.time());meta=dict(c=8,r=5,e=[],p=[],m=['quests'],i=['Cone Thrower','Mage','Undead'],j=[['session_start',now-20,'Tirisfal Glades','Deathknell',3]])
  snap=expand_snapshot([now,3,'Tirisfal Glades','70862',1,[],meta])
  self.assertEqual(snap['name'],'Cone Thrower');self.assertEqual(snap['class_name'],'Mage');self.assertTrue(snap['journey_partial'])
  self.assertEqual(snap['recent_journey'][0]['subzone'],'Deathknell')
  meta['j'][0][0]='note'
  with self.assertRaises(ValueError):expand_snapshot([now,3,'Tirisfal Glades','70862',1,[],meta])
  meta['j']=[];meta['i'][0]='x'*101
  with self.assertRaises(ValueError):expand_snapshot([now,3,'Tirisfal Glades','70862',1,[],meta])

if __name__=='__main__':unittest.main()
