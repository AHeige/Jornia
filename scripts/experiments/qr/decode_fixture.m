// Independent platform decoder used by the Lua encoder interoperability test.
// Reads a generated test PNG, never a screen or player file.
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <Vision/Vision.h>
int main(int argc,const char **argv){@autoreleasepool{
 if(argc!=2)return 2;
 NSBitmapImageRep *image=[[NSBitmapImageRep alloc]initWithData:[NSData dataWithContentsOfFile:@(argv[1])]];
 if(!image)return 2;
 VNDetectBarcodesRequest *request=[VNDetectBarcodesRequest new];request.symbologies=@[VNBarcodeSymbologyQR];
 BOOL ok=[[[VNImageRequestHandler alloc]initWithCGImage:image.CGImage options:@{}]performRequests:@[request] error:nil];
 if(!ok||request.results.count!=1)return 1;
 NSString *text=((VNBarcodeObservation *)request.results.firstObject).payloadStringValue;
 if(!text)return 1;
 NSData *data=[text dataUsingEncoding:NSUTF8StringEncoding];fwrite(data.bytes,1,data.length,stdout);return 0;
}}
