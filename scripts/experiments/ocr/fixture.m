#import <AppKit/AppKit.h>
int main(int argc,const char *argv[]){@autoreleasepool{
 NSBitmapImageRep *rep=[[NSBitmapImageRep alloc]initWithBitmapDataPlanes:NULL pixelsWide:1240 pixelsHigh:920 bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0];
 [NSGraphicsContext saveGraphicsState];[NSGraphicsContext setCurrentContext:[NSGraphicsContext graphicsContextWithBitmapImageRep:rep]];
 [[NSColor colorWithCalibratedRed:.025 green:.035 blue:.055 alpha:1]setFill];NSRectFill(NSMakeRect(0,0,1240,920));
 [[NSColor magentaColor]setFill];for(int i=0;i<4;i++)NSRectFill(NSMakeRect(i&1?1228:0,i&2?908:0,12,12));
 NSDictionary *attrs=@{NSFontAttributeName:[NSFont monospacedSystemFontOfSize:28 weight:NSFontWeightRegular],NSForegroundColorAttributeName:NSColor.whiteColor};
 NSString *text=[NSString stringWithUTF8String:argv[2]];
 [text drawInRect:NSMakeRect(40,480,1160,200) withAttributes:attrs];
 [NSGraphicsContext restoreGraphicsState];[[rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}]writeToFile:[NSString stringWithUTF8String:argv[1]] atomically:YES];return 0;
}}
