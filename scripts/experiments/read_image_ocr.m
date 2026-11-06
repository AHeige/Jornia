#import <Foundation/Foundation.h>
#import <Vision/Vision.h>

// File-only local OCR. No screen capture, input hooks, game access or network.
int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 2) return 2;
        NSTimeInterval start = NSProcessInfo.processInfo.systemUptime;
        NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:argv[1]]];
        VNRecognizeTextRequest *request = [[VNRecognizeTextRequest alloc] init];
        request.recognitionLevel = VNRequestTextRecognitionLevelAccurate;
        request.usesLanguageCorrection = NO;
        request.recognitionLanguages = @[@"en-US"];
        VNImageRequestHandler *handler = [[VNImageRequestHandler alloc] initWithURL:url options:@{}];
        NSError *error = nil;
        if (![handler performRequests:@[request] error:&error]) {
            fprintf(stderr, "OCR failed: %s\n", error.localizedDescription.UTF8String);
            return 1;
        }
        NSMutableArray *lines = [NSMutableArray array];
        for (VNRecognizedTextObservation *observation in request.results) {
            VNRecognizedText *candidate = [observation topCandidates:1].firstObject;
            if (!candidate) continue;
            CGRect b = observation.boundingBox;
            [lines addObject:@{@"text":candidate.string, @"confidence":@(candidate.confidence),
                @"bounds":@{@"x":@(b.origin.x),@"y":@(b.origin.y),@"width":@(b.size.width),@"height":@(b.size.height)}}];
        }
        NSDictionary *result = @{@"engine":@"macOS Vision, local file only",@"elapsed_seconds":@(NSProcessInfo.processInfo.systemUptime-start),@"lines":lines};
        NSData *data = [NSJSONSerialization dataWithJSONObject:result options:NSJSONWritingPrettyPrinted error:&error];
        if (!data) return 1;
        fwrite(data.bytes,1,data.length,stdout);
        return 0;
    }
}
