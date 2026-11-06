import pathlib
import tempfile
import unittest
import zlib
from block import packet,receipt,validate
from bus import Carrier,check,restore

class BlockTest(unittest.TestCase):
    def test_packet_boundaries(self):
        for payload in [b'',b'Hi',b'\0\xff\x55\xaa','åäö'.encode(),b'x'*1024]:
            wire=packet(payload,'12345678','12715')
            self.assertEqual(len(wire)%8,0)
            self.assertEqual(wire[11:11+len(payload)],payload)
            end=11+len(payload)
            self.assertEqual(int.from_bytes(wire[end:end+4],'big'),zlib.adler32(wire[:end]))
            self.assertTrue(all(x==0 for x in wire[end+4:]))
        with self.assertRaises(ValueError):packet(b'x'*1025,'12345678','12715')

    def test_checked_receipts_cannot_skip_or_repeat_blocks(self):
        payload=b'Hi';wire=packet(payload,'12345678','12715')
        raw=f'WBACK 12345678 12715 001 0008 {zlib.adler32(wire[:8]):010d}'
        got=receipt([raw+f' {check(raw):05d}'])
        self.assertEqual(validate(got,'12345678','12715',1,wire,payload,None),'advance')
        self.assertEqual(receipt([(raw+f' {check(raw):05d}').replace(' ', '')]),got)
        self.assertIsNotNone(receipt(['WBREADY 74045032 12715 000 0008 000000000043253']))
        self.assertEqual(validate(got,'12345678','12715',2,wire,payload,None),'ignore')
        self.assertEqual(validate(got,'99999999','12715',1,wire,payload,None),'ignore')
        with self.assertRaises(RuntimeError):validate(dict(got,seq=3),'12345678','12715',1,wire,payload,None)
        self.assertIsNone(receipt([(raw+f' {check(raw):05d}').replace('0008','0009')]))

    def test_wide_bank_is_reusable_and_recoverable(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=pathlib.Path(tmp);(root/'WTF').mkdir();addons=root/'Interface/AddOns'
            for n in range(1,65):
                name=f'WispOCRInbox{n:03d}';p=addons/name/(name+'_Camelot.toc');p.parent.mkdir(parents=True)
                p.write_text('## LoadOnDemand: 1\n## Dependencies: WispOCR\n')
            p=addons/'WispBridgeLab/wispreturn_pulse.wav';p.parent.mkdir();p.write_bytes(b'')
            carrier=Carrier(root,width=64);original={p:p.read_bytes() for p in carrier.paths}
            try:
                for value in [0,2**64-1,0xAA55AA55AA55AA55,0]:
                    carrier.publish(value)
                    self.assertEqual(sum(1<<i for i,p in enumerate(carrier.data) if p.exists()),value)
                restore(root,width=64)
                for p,data in original.items():self.assertEqual(p.read_bytes(),data)
            finally:carrier.restore()

if __name__=='__main__':unittest.main()
