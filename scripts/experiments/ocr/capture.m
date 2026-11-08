#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <Vision/Vision.h>
#import <CoreGraphics/CoreGraphics.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <CoreImage/CoreImage.h>
#import <CoreMedia/CoreMedia.h>
#include <unistd.h>

static void emit(NSDictionary *value) {
 NSData *d=[NSJSONSerialization dataWithJSONObject:value options:0 error:nil];
 fwrite(d.bytes,1,d.length,stdout); puts(""); fflush(stdout);
}
// ScreenCaptureKit may complete on the main run loop. Blocking that thread
// indefinitely can strand a screenshot request, so pump it with a deadline.
static BOOL awaitCapture(dispatch_semaphore_t ready,double seconds) {
 CFAbsoluteTime until=CFAbsoluteTimeGetCurrent()+seconds;
 while(dispatch_semaphore_wait(ready,DISPATCH_TIME_NOW)!=0){
  if(CFAbsoluteTimeGetCurrent()>until)return NO;
  [[NSRunLoop currentRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:.01]];
 }
 return YES;
}
static BOOL fastMode=NO;
static VNRecognizeTextRequest *request(void) {
 VNRecognizeTextRequest *r=[VNRecognizeTextRequest new]; r.recognitionLevel=fastMode?VNRequestTextRecognitionLevelFast:VNRequestTextRecognitionLevelAccurate;
 r.usesLanguageCorrection=NO; r.recognitionLanguages=@[@"en-US"]; return r;
}
// A four-corner pure-magenta frame marks the explicitly opened Jornia panel.
// Only this rectangle reaches Vision. Full-window pixels stay in memory.
static BOOL cardMode=NO;
static BOOL diagnostic=NO;
static NSString *saveCard=nil;
static CGRect panel(CGImageRef image) {
 size_t w=CGImageGetWidth(image),h=CGImageGetHeight(image),stride=w*4;
 if(w>10000||h>10000)return CGRectNull;
 unsigned char *data=calloc(h,stride); if(!data)return CGRectNull;
 CGColorSpaceRef cs=CGColorSpaceCreateDeviceRGB();
 CGContextRef ctx=CGBitmapContextCreate(data,w,h,8,stride,cs,kCGImageAlphaPremultipliedLast|kCGBitmapByteOrder32Big);
 CGColorSpaceRelease(cs); if(!ctx){free(data);return CGRectNull;}
 CGContextDrawImage(ctx,CGRectMake(0,0,w,h),image);
 size_t minx=w,miny=h,maxx=0,maxy=0,count=0;
 for(size_t y=0;y<h;y+=2)for(size_t x=0;x<w;x+=2){unsigned char *p=data+y*stride+x*4;
  if(p[0]>245&&p[1]<12&&p[2]>245){minx=MIN(minx,x);miny=MIN(miny,y);maxx=MAX(maxx,x);maxy=MAX(maxy,y);count++;}
 }
 BOOL corners=YES;
 if(minx<w&&miny<h)for(int i=0;i<4;i++){
  size_t x=(i&1)?maxx:minx,y=(i&2)?maxy:miny;BOOL found=NO;
  for(size_t yy=(y>8?y-8:0);yy<MIN(h,y+9);yy++)for(size_t xx=(x>8?x-8:0);xx<MIN(w,x+9);xx++){
   unsigned char *p=data+yy*stride+xx*4;if(p[0]>245&&p[1]<12&&p[2]>245)found=YES;
  }if(!found)corners=NO;
 }
 CGContextRelease(ctx);free(data);
 if(diagnostic) emit(@{@"diagnostic":@YES,@"width":@(w),@"height":@(h),@"magenta":@(count),@"minx":@(minx),@"miny":@(miny),@"maxx":@(maxx),@"maxy":@(maxy),@"corners":@(corners)});
 if(!corners||count<(cardMode?4:12)||maxx<=minx||maxy<=miny||maxx-minx<300||maxy-miny<200)return CGRectNull;
 // All four corners must be present: reject unrelated isolated magenta pixels.
 // Read only the outbox band, excluding the answer, input and other game chat.
 CGFloat height=maxy-miny;
 return CGRectIntersection(cardMode?CGRectMake(minx+8,miny+28,maxx-minx-16,height-60):CGRectMake(minx+8,miny+height*.22,maxx-minx-16,height*.33),CGRectMake(0,0,w,h));
}
// QR is an explicit alternate lab mode. No text recognition or glyph repair.
static BOOL qrMode=NO;
static CGRect qrBox;
static int qrMisses=0;
static NSDictionary *readQR(CGImageRef image) {
 CGRect bounds=CGRectMake(0,0,CGImageGetWidth(image),CGImageGetHeight(image));
 CGRect box=CGRectIsNull(qrBox)?bounds:CGRectIntersection(bounds,qrBox);
 if(CGRectIsNull(box)||CGRectIsEmpty(box))box=bounds;
 CGImageRef crop=CGImageCreateWithImageInRect(image,box);
 VNDetectBarcodesRequest *r=[VNDetectBarcodesRequest new];r.symbologies=@[VNBarcodeSymbologyQR];
 CFAbsoluteTime start=CFAbsoluteTimeGetCurrent();
 BOOL ok=[[[VNImageRequestHandler alloc]initWithCGImage:crop options:@{}]performRequests:@[r] error:nil];
 NSMutableArray *values=[NSMutableArray array];
 if(ok)for(VNBarcodeObservation *o in r.results){NSString *text=o.payloadStringValue;
  if(!text||text.length>1800||!([text hasPrefix:@"JORNIAQR1\n"]||[text hasPrefix:@"WBACK "]||[text hasPrefix:@"WBDONE "]||[text hasPrefix:@"WBFAIL "]))continue;
  [values addObject:text];CGRect b=o.boundingBox;
  CGRect located=CGRectMake(box.origin.x+b.origin.x*box.size.width,box.origin.y+(1-CGRectGetMaxY(b))*box.size.height,b.size.width*box.size.width,b.size.height*box.size.height);
  // Leave quiet-zone/resize room; subsequent decoding is confined to this tile.
  qrBox=CGRectIntersection(bounds,CGRectInset(located,-36,-36));qrMisses=0;
 }
 if(!values.count&&++qrMisses>=5){qrBox=CGRectNull;qrMisses=0;}
 NSDictionary *result=@{@"qr":values,@"ocr_ms":@((CFAbsoluteTimeGetCurrent()-start)*1000),@"crop_width":@(CGImageGetWidth(crop)),@"crop_height":@(CGImageGetHeight(crop))};
 CGImageRelease(crop);return result;
}
// Keep one current frame; no screenshot files or frame queue are retained.
@interface WispFrames:NSObject<SCStreamOutput,SCStreamDelegate>
@property(nonatomic) CVPixelBufferRef latest;
@property(nonatomic) BOOL failed;
@end
@implementation WispFrames
-(void)stream:(SCStream*)stream didOutputSampleBuffer:(CMSampleBufferRef)sample ofType:(SCStreamOutputType)type {
 if(type!=SCStreamOutputTypeScreen||!CMSampleBufferIsValid(sample))return;
 CVPixelBufferRef buffer=CMSampleBufferGetImageBuffer(sample);if(!buffer)return;
 @synchronized(self){if(_latest)CVPixelBufferRelease(_latest);_latest=CVPixelBufferRetain(buffer);}
}
-(void)stream:(SCStream*)stream didStopWithError:(NSError*)error {self.failed=YES;}
-(CVPixelBufferRef)take { @synchronized(self){CVPixelBufferRef value=_latest;_latest=NULL;return value;} }
-(void)dealloc{if(_latest)CVPixelBufferRelease(_latest);}
@end
int main(int argc,const char *argv[]) {
 @autoreleasepool {
  qrBox=CGRectNull;
  const char *file=NULL;for(int i=1;i<argc;i++){if(!strcmp(argv[i],"--qr"))qrMode=YES;else if(!strcmp(argv[i],"--card"))cardMode=YES;else if(!strcmp(argv[i],"--fast"))fastMode=YES;else if(!strcmp(argv[i],"--diagnostic"))diagnostic=YES;else if(!strcmp(argv[i],"--save-card")&&i+1<argc)saveCard=[NSString stringWithUTF8String:argv[++i]];else if(!strcmp(argv[i],"--file")&&i+1<argc)file=argv[++i];}
  if(file){
   NSBitmapImageRep *rep=[[NSBitmapImageRep alloc]initWithData:[NSData dataWithContentsOfFile:[NSString stringWithUTF8String:file]]];
   if(!rep)return 2;if(qrMode){emit(readQR(rep.CGImage));return 0;}CGRect box=panel(rep.CGImage);if(CGRectIsNull(box)){emit(@{@"error":@"panel_not_found"});return 4;}
   CGImageRef crop=CGImageCreateWithImageInRect(rep.CGImage,box);VNRecognizeTextRequest *r=request();
   CFAbsoluteTime start=CFAbsoluteTimeGetCurrent();
   [[[VNImageRequestHandler alloc]initWithCGImage:crop options:@{}] performRequests:@[r] error:nil];
   NSMutableArray *lines=[NSMutableArray array];for(VNRecognizedTextObservation *o in r.results){VNRecognizedText *c=[o topCandidates:1].firstObject;if(c)[lines addObject:c.string];}
   emit(@{@"lines":lines,@"ocr_ms":@((CFAbsoluteTimeGetCurrent()-start)*1000),@"crop_width":@(CGImageGetWidth(crop)),@"crop_height":@(CGImageGetHeight(crop))});CGImageRelease(crop);return 0;
  }
  if(argc==2 && !strcmp(argv[1],"--permission")){BOOL ok=CGPreflightScreenCaptureAccess()||CGRequestScreenCaptureAccess();emit(@{@"permission":@(ok)});return ok?0:3;}
  if(!CGPreflightScreenCaptureAccess()){emit(@{@"error":@"screen_permission_required"});return 3;}
  // Warm Vision before accepting an Enter. No user screenshot is needed.
  unsigned char pixels[256*64*4];memset(pixels,255,sizeof(pixels));
  CGColorSpaceRef cs=CGColorSpaceCreateDeviceRGB();CGContextRef ctx=CGBitmapContextCreate(pixels,256,64,8,256*4,cs,kCGImageAlphaPremultipliedLast);
  CGImageRef blank=CGBitmapContextCreateImage(ctx); VNRecognizeTextRequest *r=request();
  [[[VNImageRequestHandler alloc]initWithCGImage:blank options:@{}] performRequests:@[r] error:nil];
  CGImageRelease(blank);CGContextRelease(ctx);CGColorSpaceRelease(cs);emit(@{@"ready":@YES});
  CIContext *images=[CIContext contextWithOptions:@{kCIContextUseSoftwareRenderer:@NO}];
  while(1){@autoreleasepool{
   __block SCShareableContent *content=nil;dispatch_semaphore_t available=dispatch_semaphore_create(0);
   [SCShareableContent getShareableContentExcludingDesktopWindows:YES onScreenWindowsOnly:YES completionHandler:^(SCShareableContent *c,NSError *e){content=c;dispatch_semaphore_signal(available);}];
   if(!awaitCapture(available,3)){usleep(300000);continue;}
   SCWindow *window=nil;
   for(SCWindow *w in content.windows)if([w.owningApplication.bundleIdentifier isEqualToString:@"com.blizzard.worldofwarcraft"]&&w.windowLayer==0){window=w;break;}
   if(!window){usleep(300000);continue;}
   WispFrames *frames=[WispFrames new];
   SCContentFilter *filter=[[SCContentFilter alloc]initWithDesktopIndependentWindow:window];
   SCStreamConfiguration *config=[SCStreamConfiguration new];config.width=window.frame.size.width*2;config.height=window.frame.size.height*2;config.showsCursor=NO;config.queueDepth=3;config.minimumFrameInterval=CMTimeMake(1,5);config.pixelFormat=kCVPixelFormatType_32BGRA;
   SCStream *stream=[[SCStream alloc]initWithFilter:filter configuration:config delegate:frames];
   NSError *setup=nil;dispatch_queue_t output=dispatch_queue_create("wisp.local.frames",DISPATCH_QUEUE_SERIAL);
   if(![stream addStreamOutput:frames type:SCStreamOutputTypeScreen sampleHandlerQueue:output error:&setup]){emit(@{@"error":@"capture_setup_failed"});return 5;}
   __block BOOL running=NO;dispatch_semaphore_t begun=dispatch_semaphore_create(0);
   [stream startCaptureWithCompletionHandler:^(NSError *error){running=error==nil;dispatch_semaphore_signal(begun);}];
   if(!awaitCapture(begun,5)||!running){emit(@{@"error":@"capture_start_failed"});return 5;}
   CFAbsoluteTime lastFrame=CFAbsoluteTimeGetCurrent(),tick=0;
   while(!frames.failed&&CFAbsoluteTimeGetCurrent()-lastFrame<10){@autoreleasepool{
    [[NSRunLoop currentRunLoop]runUntilDate:[NSDate dateWithTimeIntervalSinceNow:.05]];
    CVPixelBufferRef pixel=[frames take];if(!pixel)continue;lastFrame=CFAbsoluteTimeGetCurrent();if(lastFrame-tick>1){tick=lastFrame;emit(@{@"capture_tick":@YES});}
    CIImage *ci=[CIImage imageWithCVPixelBuffer:pixel];CGImageRef image=[images createCGImage:ci fromRect:ci.extent];CVPixelBufferRelease(pixel);if(!image)continue;
    if(qrMode){emit(readQR(image));CGImageRelease(image);continue;}
    CGRect box=panel(image);
    if(!CGRectIsNull(box)){
     CGImageRef crop=CGImageCreateWithImageInRect(image,box);
     if(saveCard){if([[NSFileManager defaultManager]fileExistsAtPath:saveCard])return 6;NSBitmapImageRep *rep=[[NSBitmapImageRep alloc]initWithCGImage:crop];NSData *png=[rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];[png writeToFile:saveCard atomically:YES];emit(@{@"saved_card":@YES});CGImageRelease(crop);CGImageRelease(image);return 0;}
     CFAbsoluteTime start=CFAbsoluteTimeGetCurrent();
     BOOL ok=[[[VNImageRequestHandler alloc]initWithCGImage:crop options:@{}]performRequests:@[r]error:nil];
     NSMutableArray *lines=[NSMutableArray array],*candidates=[NSMutableArray array];
     if(ok)for(VNRecognizedTextObservation *o in r.results){NSMutableArray *row=[NSMutableArray array];for(VNRecognizedText *c in [o topCandidates:3]){[row addObject:c.string];}if(row.count){[lines addObject:row.firstObject];[candidates addObject:row];}}
     emit(@{@"lines":lines,@"candidates":candidates,@"ocr_ms":@((CFAbsoluteTimeGetCurrent()-start)*1000)});CGImageRelease(crop);
    }
    CGImageRelease(image);
   }}
   dispatch_semaphore_t ended=dispatch_semaphore_create(0);[stream stopCaptureWithCompletionHandler:^(NSError *error){dispatch_semaphore_signal(ended);}];awaitCapture(ended,3);
  }}
 }
}
