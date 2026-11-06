import pathlib
import tempfile
import unittest
import zlib
from bus import Carrier,packet,receipt,check,restore

class BusTest(unittest.TestCase):
    def test_packet_binary_edges_and_utf8(self):
        for text in [b'',b'\0\x55\xaa\xff','åäö'.encode()]:
            data=packet(text)
            self.assertEqual(data[3:-4],text)
            self.assertEqual(int.from_bytes(data[-4:],'big'),zlib.adler32(data[:-4]))
        with self.assertRaises(ValueError):packet(b'x'*97)

    def test_damaged_ocr_receipt_rejected(self):
        raw='WISPACK 12345678 12715 003 255 0000123456'
        line=raw+f' {check(raw):05d}'
        self.assertEqual(receipt([line])['value'],255)
        self.assertEqual(receipt([line.replace(' ','\n',1)])['seq'],3)
        self.assertIsNone(receipt([line.replace('255','254')]))

    def test_same_carriers_reused_and_restored(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=pathlib.Path(tmp);(root/'WTF').mkdir()
            addons=root/'Interface/AddOns'
            for n in range(81,89):
                name=f'WispOCRInbox{n:03d}';p=addons/name/(name+'_Camelot.toc');p.parent.mkdir(parents=True)
                p.write_text('## LoadOnDemand: 1\n## Dependencies: WispOCR\n')
            clock=addons/'WispBridgeLab/wispreturn_pulse.wav';clock.parent.mkdir();clock.write_bytes(b'')
            carrier=Carrier(root);original={p:p.read_bytes() for p in carrier.paths}
            try:
                for value in [0,255,85,170,0,255]:
                    carrier.publish(value)
                    self.assertEqual(sum(1<<i for i,p in enumerate(carrier.data) if p.exists()),value)
                # Simulate a hard exit and recover without trusting a path manifest.
                restore(root)
                for p,data in original.items():self.assertEqual(p.read_bytes(),data)
            finally:carrier.restore()

if __name__=='__main__':unittest.main()
