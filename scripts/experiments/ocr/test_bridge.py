import json
import pathlib
import tempfile
import unittest
from unittest.mock import patch
import bridge


class ProtocolTest(unittest.TestCase):
    nonce = '23456789'
    ident = nonce + '1793960000' + '000001'
    guid = 'Player-1234-ABCDEF'

    def card(self, q='Where should I go next?', binding=None):
        binding = binding or f'{bridge.checksum(self.guid):05d}'
        digest = bridge.checksum(self.ident + binding + q)
        return [f'WISPBEGIN {self.ident} {binding} {digest:05d}', q, 'WISPEND']

    def test_correlated_card_and_damage_rejection(self):
        valid = bridge.parse(self.card(), self.nonce, {self.guid: 'local-char'})
        self.assertEqual(valid['question'], 'Where should I go next?')
        split = self.card(); fields = split[0].split(); split[0] = ' '.join(fields[:-1]) + '\n' + fields[-1]
        self.assertIsNotNone(bridge.parse(split, self.nonce, {self.guid: 'local-char'}))
        damaged = self.card(); damaged[1] = 'Where should I go next !'
        self.assertIsNone(bridge.parse(damaged, self.nonce, {self.guid: 'local-char'}))
        self.assertIsNone(bridge.parse(self.card()[:-1], self.nonce, {self.guid: 'local-char'}))
        self.assertIsNone(bridge.parse(self.card(), '99999999', {self.guid: 'local-char'}))
        self.assertIsNone(bridge.parse(self.card(), self.nonce, {}))

    def test_real_file_return_preserves_text_as_data_and_correlates_api(self):
        packet = bridge.parse(self.card(), self.nonce, {self.guid: 'local-char'})
        calls = []
        def api(session, path, data=None):
            calls.append((path, data))
            if path == '/local/ask':
                self.assertEqual(data['mutation_id'], 'ocr-' + self.ident)
                self.assertEqual(data['character_id'], 'local-char')
                self.assertTrue(data['ai'])
                self.assertNotIn('context_ids', data)
                return {'status': 'pending'}
            return {'status': 'completed', 'result': {'text': 'Reply "quotes" \\ åäö', 'route': 'test-only'}}
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            for n in range(1, 97):
                (root / f'WispOCRInbox{n:03d}').mkdir()
            evidence = {'ocr_ms': 200}
            with patch.object(bridge, 'api', api), patch.object(bridge.time, 'sleep'):
                bridge.complete({}, root, packet, evidence)
            self.assertEqual(len(calls), 2)
            body = (root / 'WispOCRInbox096/Inbox.lua').read_text()
            self.assertIn(self.ident, body)
            self.assertIn('"done"', body)
            self.assertNotIn('åäö', body)
            encoded = body.split('"')[-2]
            self.assertIn('Reply "quotes" \\ åäö', bytes.fromhex(encoded).decode())
            self.assertEqual(evidence['status'], 'reply_files_written')

    def test_failure_is_delivered_without_retry(self):
        packet = bridge.parse(self.card(), self.nonce, {self.guid: 'local-char'})
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            for n in range(1, 97): (root / f'WispOCRInbox{n:03d}').mkdir()
            with patch.object(bridge, 'api', side_effect=TimeoutError) as api:
                bridge.complete({}, root, packet, {'ocr_ms': 200})
            self.assertEqual(api.call_count, 1)
            self.assertIn('"error"', (root / 'WispOCRInbox001/Inbox.lua').read_text())


if __name__ == '__main__': unittest.main()

class UnicodeSnapshotTest(unittest.TestCase):
    def packet(self, q='次は何をすればいい？', snapshot=None):
        wire=json.dumps({'q':q,'s':snapshot},ensure_ascii=False,separators=(',',':'))
        ident=ProtocolTest.ident; binding=f'{bridge.checksum(ProtocolTest.guid):05d}'
        encoded=wire.encode().hex()
        lines=[f'WISPBEGIN2 {ident} {binding} {bridge.checksum(ident+binding+wire):05d}',encoded[:80],encoded[80:],'WISPEND']
        return bridge.parse(lines,ProtocolTest.nonce,{ProtocolTest.guid:'char'})
    def test_unicode_and_explicit_snapshot(self):
        for q in ['次は何をすればいい？','إلى أين أذهب؟','Vad gör jag nu?','¿Qué hago ahora?','Куда идти?','quote " backslash \\ pipe |']:
            p=self.packet(q);self.assertEqual(p['question'],q);self.assertNotIn('snapshot',p)
        snapshot=[int(bridge.time.time()),16,'灰谷','70790',1,[[123,'Quest',1,1]]]
        p=self.packet(snapshot=snapshot);self.assertTrue(p['snapshot']['quests'][0]['waypoint'])
        result={'actions':[{'id':'navigate:quest:123','kind':'show_quest_waypoint','quest_id':123,'expires_at':snapshot[0]+90}],'evidence_ids':['state:quest:123']}
        self.assertIsNotNone(bridge.navigation(p,result))
        result['actions'][0]['quest_id']=999;self.assertIsNone(bridge.navigation(p,result))
        snapshot[0]-=100;self.assertIsNone(self.packet(snapshot=snapshot))
    def test_invalid_shapes(self):
        self.assertIsNone(self.packet(snapshot=[True,16,'z','b',0,[]]))
        self.assertIsNone(self.packet(q='界'*60))
        self.assertIsNone(self.packet(snapshot=[int(bridge.time.time()),16,'z','b',0,[[1,'x',1,1],[1,'x',1,1]]]))
