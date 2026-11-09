#import <Cocoa/Cocoa.h>

@interface JorniaApp : NSObject <NSApplicationDelegate, NSWindowDelegate>
@property NSWindow *window;
@property NSTextField *rootLabel, *status, *detail;
@property NSButton *start, *stop, *install, *choose, *permission, *journey;
@property NSTask *worker;
@property NSMutableData *pending;
@property NSString *root, *localURL;
@property BOOL quitting, failed;
@end

@implementation JorniaApp
- (NSString *)resources { return NSBundle.mainBundle.resourcePath; }
- (NSTextField *)label:(NSString *)text frame:(NSRect)frame size:(CGFloat)size {
    NSTextField *label=[NSTextField wrappingLabelWithString:text];
    label.frame=frame; label.font=[NSFont systemFontOfSize:size];
    label.textColor=NSColor.labelColor; [self.window.contentView addSubview:label]; return label;
}
- (NSButton *)button:(NSString *)title action:(SEL)action frame:(NSRect)frame {
    NSButton *button=[NSButton buttonWithTitle:title target:self action:action];
    button.frame=frame; button.bezelStyle=NSBezelStyleRounded;
    [self.window.contentView addSubview:button]; return button;
}
- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    NSApp.appearance=[NSAppearance appearanceNamed:NSAppearanceNameDarkAqua];
    self.window=[[NSWindow alloc] initWithContentRect:NSMakeRect(0,0,660,490)
        styleMask:NSWindowStyleMaskTitled|NSWindowStyleMaskClosable|NSWindowStyleMaskMiniaturizable
        backing:NSBackingStoreBuffered defer:NO];
    self.window.title=@"Jornia"; self.window.delegate=self;
    self.window.backgroundColor=[NSColor colorWithRed:.055 green:.075 blue:.09 alpha:1];
    [self label:@"YOUR JOURNEY IN AZEROTH" frame:NSMakeRect(32,436,590,22) size:11].textColor=[NSColor colorWithRed:.85 green:.71 blue:.43 alpha:1];
    [self label:@"A companion for the road ahead." frame:NSMakeRect(32,378,596,49) size:27];
    [self label:@"Your character. Your quests. Your story.\nAsk Jornia in WoW; keep your journey on this Mac." frame:NSMakeRect(32,324,596,46) size:14];
    self.root=[[NSUserDefaults standardUserDefaults] stringForKey:@"ForeverRoot"] ?: @"";
    [self label:@"WORLD OF WARCRAFT FOREVER" frame:NSMakeRect(32,280,500,20) size:11];
    self.rootLabel=[self label:self.root.length?self.root:@"Choose the game folder containing WTF and Interface." frame:NSMakeRect(32,241,450,35) size:12];
    self.choose=[self button:@"Choose folder…" action:@selector(chooseFolder:) frame:NSMakeRect(500,245,130,32)];
    self.install=[self button:@"Install / update addon" action:@selector(installAddon:) frame:NSMakeRect(26,202,190,32)];
    self.permission=[self button:@"Allow local screen capture…" action:@selector(allowCapture:) frame:NSMakeRect(220,202,226,32)];
    self.status=[self label:@"Ready to connect" frame:NSMakeRect(32,158,596,25) size:17];
    self.detail=[self label:@"Capture stays on this Mac. Only your question and approved character context are sent to AI. Keep the in-game chat visible while waiting." frame:NSMakeRect(32,89,596,61) size:12];
    self.detail.textColor=NSColor.secondaryLabelColor;
    self.start=[self button:@"Start in-game chat" action:@selector(startChat:) frame:NSMakeRect(26,32,180,36)];
    self.start.keyEquivalent=@"\r";
    self.journey=[self button:@"Open local journey" action:@selector(openJourney:) frame:NSMakeRect(216,32,180,36)]; self.journey.enabled=NO;
    self.stop=[self button:@"Stop" action:@selector(stopChat:) frame:NSMakeRect(520,32,110,36)]; self.stop.enabled=NO;
    NSMenu *menu=[NSMenu new]; NSMenuItem *appItem=[NSMenuItem new]; [menu addItem:appItem];
    NSMenu *appMenu=[NSMenu new]; [appMenu addItemWithTitle:@"Quit Jornia" action:@selector(terminate:) keyEquivalent:@"q"]; appItem.submenu=appMenu; NSApp.mainMenu=menu;
    [self.window center]; [self.window makeKeyAndOrderFront:nil]; [NSApp activateIgnoringOtherApps:YES];
}
- (void)chooseFolder:(id)sender {
    NSOpenPanel *panel=[NSOpenPanel openPanel]; panel.canChooseDirectories=YES; panel.canChooseFiles=NO; panel.allowsMultipleSelection=NO;
    panel.message=@"Choose your Forever client folder containing WTF and Interface.";
    [panel beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse response){
        if(response!=NSModalResponseOK)return;
        NSString *path=panel.URL.path;
        if(![[NSFileManager defaultManager] fileExistsAtPath:[path stringByAppendingPathComponent:@"WTF"]]){
            self.status.stringValue=@"That is not the game folder"; self.detail.stringValue=@"Choose the client folder containing WTF and Interface, usually named _forever_."; return;
        }
        self.root=path; self.rootLabel.stringValue=path;
        [[NSUserDefaults standardUserDefaults] setObject:path forKey:@"ForeverRoot"];
    }];
}
- (NSDictionary *)pythonEnvironment {
    NSMutableDictionary *env=[NSProcessInfo.processInfo.environment mutableCopy];
    for(NSString *key in [env.allKeys copy])if([key hasPrefix:@"PYTHON"] || [key hasPrefix:@"DYLD_"])[env removeObjectForKey:key];
    env[@"PYTHONHOME"]=[[self resources] stringByAppendingPathComponent:@"Runtime"];
    env[@"PYTHONNOUSERSITE"]=@"1"; env[@"PYTHONUNBUFFERED"]=@"1"; env[@"PYTHONDONTWRITEBYTECODE"]=@"1";
    env[@"SSL_CERT_FILE"]=[[self resources] stringByAppendingPathComponent:@"cacert.pem"];
    return env;
}
- (void)handleLine:(NSData *)line {
    NSDictionary *state=[NSJSONSerialization JSONObjectWithData:line options:0 error:nil];
    if(![state isKindOfClass:NSDictionary.class])return;
    NSString *name=state[@"state"], *message=state[@"message"];
    if([name isEqual:@"error"] || [name isEqual:@"permission"])self.failed=YES;
    if(self.failed && ([name isEqual:@"stopping"] || [name isEqual:@"stopped"]))return;
    if([message isKindOfClass:NSString.class])self.detail.stringValue=message;
    NSDictionary *titles=@{@"starting":@"Connecting…", @"waiting":@"Waiting for WoW", @"ready":@"In-game chat ready", @"attention":@"Check the connection", @"permission":@"Screen capture needs permission", @"stopping":@"Stopping safely…", @"stopped":@"Disconnected", @"installed":@"Addon installed", @"error":@"Connection needs attention"};
    self.status.stringValue=titles[name] ?: @"Jornia";
    NSString *url=state[@"url"];
    if([url isKindOfClass:NSString.class] && [url hasPrefix:@"http://127.0.0.1:"]){self.localURL=url;self.journey.enabled=YES;}
}
- (void)launch:(NSString *)action {
    if(self.worker.running)return;
    if(!self.root.length){[self chooseFolder:nil];return;}
    self.failed=NO; self.pending=[NSMutableData data];
    self.status.stringValue=[action isEqual:@"install"]?@"Installing addon…":@"Connecting…";
    self.start.enabled=self.choose.enabled=self.install.enabled=self.permission.enabled=NO;
    self.stop.enabled=[action isEqual:@"start"];
    NSTask *task=[NSTask new]; self.worker=task;
    task.executableURL=[NSURL fileURLWithPath:[[self resources] stringByAppendingPathComponent:@"Runtime/bin/python3"]];
    task.arguments=@[@"-s",@"-S",[[self resources] stringByAppendingPathComponent:@"macos/supervisor.py"],action,@"--resources",[self resources],@"--root",self.root];
    task.environment=[self pythonEnvironment];
    NSPipe *pipe=[NSPipe pipe]; task.standardOutput=pipe; task.standardError=[NSFileHandle fileHandleWithNullDevice];
    pipe.fileHandleForReading.readabilityHandler=^(NSFileHandle *handle){
        NSData *chunk=handle.availableData;
        if(!chunk.length){handle.readabilityHandler=nil;return;}
        dispatch_async(dispatch_get_main_queue(), ^{
            [self.pending appendData:chunk];
            while(YES){const char *bytes=self.pending.bytes;NSUInteger n=self.pending.length,i=0;while(i<n&&bytes[i]!='\n')i++;if(i==n)break;
                [self handleLine:[self.pending subdataWithRange:NSMakeRange(0,i)]];
                [self.pending replaceBytesInRange:NSMakeRange(0,i+1) withBytes:NULL length:0];
            }
        });
    };
    task.terminationHandler=^(NSTask *finished){dispatch_async(dispatch_get_main_queue(), ^{
        self.start.enabled=self.choose.enabled=self.install.enabled=self.permission.enabled=YES;self.stop.enabled=NO;self.journey.enabled=NO;self.localURL=nil;
        if(finished.terminationStatus && !self.failed){self.status.stringValue=@"Could not start Jornia";self.detail.stringValue=@"Your data is preserved. Check the game folder, stop any older Jornia reader, and try again.";}
        if(self.quitting)[NSApp replyToApplicationShouldTerminate:YES];
    });};
    NSError *error=nil;
    if(![task launchAndReturnError:&error]){
        pipe.fileHandleForReading.readabilityHandler=nil;
        self.start.enabled=self.choose.enabled=self.install.enabled=self.permission.enabled=YES;self.stop.enabled=NO;
        self.status.stringValue=@"Could not open the local runtime";self.detail.stringValue=@"The app bundle is incomplete or blocked by macOS. Keep security protections enabled and use a verified release.";
    }
}
- (void)startChat:(id)sender {[self launch:@"start"];}
- (void)installAddon:(id)sender {[self launch:@"install"];}
- (void)stopChat:(id)sender {self.stop.enabled=NO;[self.worker terminate];}
- (void)allowCapture:(id)sender {
    NSTask *task=[NSTask new];task.executableURL=[NSURL fileURLWithPath:[[self resources] stringByAppendingPathComponent:@"bin/jornia-capture"]];
    task.arguments=@[@"--permission"];task.standardOutput=[NSFileHandle fileHandleWithNullDevice];task.standardError=[NSFileHandle fileHandleWithNullDevice];
    task.terminationHandler=^(NSTask *done){dispatch_async(dispatch_get_main_queue(), ^{self.status.stringValue=done.terminationStatus?@"Permission is not enabled":@"Local capture permission enabled";self.detail.stringValue=done.terminationStatus?@"Allow Jornia in System Settings → Privacy & Security → Screen Recording, then start the connection again.":@"Start in-game chat when you are ready. Images stay on this Mac.";});};
    [task launchAndReturnError:nil];
}
- (void)openJourney:(id)sender {if(self.localURL)[NSWorkspace.sharedWorkspace openURL:[NSURL URLWithString:self.localURL]];}
- (NSApplicationTerminateReply)applicationShouldTerminate:(NSApplication *)sender {
    if(self.worker.running){self.quitting=YES;[self stopChat:nil];return NSTerminateLater;}
    return NSTerminateNow;
}
- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {return YES;}
@end

int main(int argc, const char *argv[]) {
    @autoreleasepool {NSApplication *app=NSApplication.sharedApplication; JorniaApp *delegate=[JorniaApp new];app.delegate=delegate;[app setActivationPolicy:NSApplicationActivationPolicyRegular];[app run];}
    return 0;
}
