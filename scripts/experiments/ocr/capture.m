#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <Vision/Vision.h>
#import <CoreGraphics/CoreGraphics.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#include <unistd.h>

static void emit(NSDictionary *value) {
 NSData *d=[NSJSONSerialization dataWithJSONObject:value options:0 error:nil];
 fwrite(d.bytes,1,d.length,stdout); puts(""); fflush(stdout);
}
static VNRecognizeTextRequest *request(void) {
 VNRecognizeTextRequest *r=[VNRecognizeTextRequest new]; r.recognitionLevel=VNRequestTextRecognitionLevelAccurate;
 r.usesLanguageCorrection=NO; r.recognitionLanguages=@[@"en-US"]; return r;
}
// A four-corner pure-magenta frame marks the explicitly opened Wisp panel.
// Only this rectangle reaches Vision. Full-window pixels stay in memory.
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
 if(!corners||count<12||maxx<=minx||maxy<=miny||maxx-minx<300||maxy-miny<200)return CGRectNull;
 // All four corners must be present: reject unrelated isolated magenta pixels.
 // Read only the outbox band, excluding the answer, input and other game chat.
 CGFloat height=maxy-miny;
 return CGRectIntersection(CGRectMake(minx+8,miny+height*.22,maxx-minx-16,height*.33),CGRectMake(0,0,w,h));
}
int main(int argc,const char *argv[]) {
 @autoreleasepool {
  if(argc==3 && !strcmp(argv[1],"--file")){
   NSBitmapImageRep *rep=[[NSBitmapImageRep alloc]initWithData:[NSData dataWithContentsOfFile:[NSString stringWithUTF8String:argv[2]]]];
   if(!rep)return 2;CGRect box=panel(rep.CGImage);if(CGRectIsNull(box)){emit(@{@"error":@"panel_not_found"});return 4;}
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
  while(1){@autoreleasepool{
   __block SCShareableContent *content=nil;
   dispatch_semaphore_t available=dispatch_semaphore_create(0);
   [SCShareableContent getShareableContentExcludingDesktopWindows:YES onScreenWindowsOnly:YES completionHandler:^(SCShareableContent *c,NSError *e){content=c;dispatch_semaphore_signal(available);}];
   if(dispatch_semaphore_wait(available,dispatch_time(DISPATCH_TIME_NOW,3*NSEC_PER_SEC))!=0){usleep(300000);continue;}
   SCWindow *window=nil;
   for(SCWindow *w in content.windows)if([w.owningApplication.bundleIdentifier isEqualToString:@"com.blizzard.worldofwarcraft"] && w.windowLayer==0){window=w;break;}
   if(window){
    CFAbsoluteTime start=CFAbsoluteTimeGetCurrent();
    SCContentFilter *filter=[[SCContentFilter alloc]initWithDesktopIndependentWindow:window];
    SCStreamConfiguration *config=[SCStreamConfiguration new];
    config.width=window.frame.size.width*2;config.height=window.frame.size.height*2;config.showsCursor=NO;
    __block CGImageRef image=NULL;
    dispatch_semaphore_t captured=dispatch_semaphore_create(0);
    [SCScreenshotManager captureImageWithFilter:filter configuration:config completionHandler:^(CGImageRef img,NSError *e){if(img)image=CGImageRetain(img);dispatch_semaphore_signal(captured);}];
    // Wait for completion before returning ownership of the captured image.
    dispatch_semaphore_wait(captured,DISPATCH_TIME_FOREVER);
    if(image){CGRect box=panel(image);if(!CGRectIsNull(box)){
     CGImageRef crop=CGImageCreateWithImageInRect(image,box);
     CFAbsoluteTime ocr=CFAbsoluteTimeGetCurrent();NSError *err=nil;
     BOOL ok=[[[VNImageRequestHandler alloc]initWithCGImage:crop options:@{}] performRequests:@[r] error:&err];
     NSMutableArray *lines=[NSMutableArray array];if(ok)for(VNRecognizedTextObservation *o in r.results){VNRecognizedText *c=[o topCandidates:1].firstObject;if(c)[lines addObject:c.string];}
     emit(@{@"lines":lines,@"ocr_ms":@((CFAbsoluteTimeGetCurrent()-ocr)*1000),@"capture_ocr_ms":@((CFAbsoluteTimeGetCurrent()-start)*1000)});
     CGImageRelease(crop);
    }CGImageRelease(image);}
   }
   usleep(300000);
  }}
 }
}
