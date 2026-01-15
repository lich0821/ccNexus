#import <Cocoa/Cocoa.h>

@interface TrayDelegate : NSObject
@property (strong, nonatomic) NSStatusItem *statusItem;
@property (strong, nonatomic) NSMenu *menu;
@end

@implementation TrayDelegate

- (void)setupTray:(NSData *)iconData {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];

        self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];

        NSImage *icon = [[NSImage alloc] initWithData:iconData];
        [icon setSize:NSMakeSize(18, 18)];
        [self.statusItem.button setImage:icon];
        [self.statusItem.button setImageScaling:NSImageScaleProportionallyDown];

        self.menu = [[NSMenu alloc] init];

        NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit | 退出"
                                                          action:@selector(quitApp:)
                                                   keyEquivalent:@""];
        [quitItem setTarget:self];
        [self.menu addItem:quitItem];

        [self.statusItem.button setTarget:self];
        [self.statusItem.button setAction:@selector(iconClicked:)];
        [[self.statusItem button] sendActionOn:NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp];
    });
}

extern void goShowWindow();
extern void goHideWindow();
extern void goQuitApp();

- (void)iconClicked:(id)sender {
    NSEvent *event = [NSApp currentEvent];
    if (event.type == NSEventTypeRightMouseUp) {
        [self.statusItem popUpStatusItemMenu:self.menu];
    } else {
        goShowWindow();
    }
}

- (void)quitApp:(id)sender {
    goQuitApp();
}

@end

static TrayDelegate *trayDelegate = nil;

void setupTray(void *iconData, int iconLen) {
    if (trayDelegate == nil) {
        trayDelegate = [[TrayDelegate alloc] init];
    }
    NSData *data = [NSData dataWithBytes:iconData length:iconLen];
    [trayDelegate setupTray:data];
}

