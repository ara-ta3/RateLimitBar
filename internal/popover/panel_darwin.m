#import <Cocoa/Cocoa.h>
#import "settings_darwin.h"
#import "status_darwin.h"

extern void rlbPanelEvent(char *action);

@interface RLBFlippedView : NSView
@end
@implementation RLBFlippedView
- (BOOL)isFlipped { return YES; }
@end

@interface RLBCard : RLBFlippedView
@end
@implementation RLBCard
- (void)drawRect:(NSRect)dirtyRect {
    [NSColor.controlBackgroundColor setFill];
    [[NSBezierPath bezierPathWithRoundedRect:self.bounds xRadius:12 yRadius:12] fill];
}
@end

@interface RLBUsageBar : NSView
@property NSInteger usedPercent;
@end
@implementation RLBUsageBar
- (BOOL)isAccessibilityElement { return YES; }
- (NSString *)accessibilityRole { return NSAccessibilityProgressIndicatorRole; }
- (id)accessibilityValue { return @(self.usedPercent); }
- (id)accessibilityMinValue { return @0; }
- (id)accessibilityMaxValue { return @100; }
- (void)drawRect:(NSRect)dirtyRect {
    [[NSColor.separatorColor colorWithAlphaComponent:0.3] setFill];
    [[NSBezierPath bezierPathWithRoundedRect:self.bounds xRadius:3 yRadius:3] fill];
    NSRect fill = self.bounds;
    fill.size.width *= MIN(100, MAX(0, self.usedPercent)) / 100.0;
    NSColor *color = self.usedPercent >= 90 ? NSColor.systemRedColor :
        (self.usedPercent >= 70 ? NSColor.systemYellowColor : NSColor.systemGreenColor);
    [color setFill];
    [[NSBezierPath bezierPathWithRoundedRect:fill xRadius:3 yRadius:3] fill];
}
@end

static NSTextField *RLBLabel(NSString *text, NSRect frame, NSFont *font, NSColor *color) {
    NSTextField *label = [NSTextField labelWithString:text];
    label.frame = frame;
    label.font = font;
    label.textColor = color;
    label.lineBreakMode = NSLineBreakByTruncatingTail;
    return label;
}

@interface RLBPanelController : NSObject
@property NSPopover *popover;
@property NSDictionary *model;
@property NSScrollView *scroll;
@property NSView *footer;
@property RLBSettingsController *settings;
- (void)render;
- (void)performAction:(id)sender;
- (void)showSettings:(NSButton *)sender;
@end

@implementation RLBPanelController
- (instancetype)init {
    if ((self = [super init])) {
        _settings = [RLBSettingsController new];
        _popover = [NSPopover new];
        _popover.behavior = NSPopoverBehaviorTransient;
        _popover.animates = YES;
        NSViewController *content = [NSViewController new];
        content.view = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 360, 400)];
        _popover.contentViewController = content;
        _scroll = [[NSScrollView alloc] initWithFrame:NSZeroRect];
        _scroll.hasVerticalScroller = YES;
        _scroll.autohidesScrollers = YES;
        _scroll.drawsBackground = NO;
        [content.view addSubview:_scroll];
        _footer = [[NSView alloc] initWithFrame:NSZeroRect];
        [content.view addSubview:_footer];
    }
    return self;
}

- (NSButton *)button:(NSDictionary *)command frame:(NSRect)frame {
    NSButton *button = [NSButton buttonWithTitle:command[@"title"] target:self action:@selector(performAction:)];
    button.frame = frame;
    button.identifier = command[@"action"];
    button.bezelStyle = NSBezelStyleRounded;
    return button;
}

- (void)render {
    [self.settings updateModel:self.model];
    NSPoint scrollPosition = self.scroll.contentView.bounds.origin;
    RLBFlippedView *document = [[RLBFlippedView alloc] initWithFrame:NSZeroRect];
    CGFloat y = 14;
    [document addSubview:RLBLabel(@"RateLimitBar", NSMakeRect(18, y, 320, 24),
        [NSFont systemFontOfSize:18 weight:NSFontWeightBold], NSColor.labelColor)];
    y += 36;
    for (NSDictionary *provider in self.model[@"providers"]) {
        NSArray *windows = provider[@"windows"];
        if ((id)windows == NSNull.null) windows = @[];
        NSString *message = provider[@"message"];
        CGFloat messageHeight = 0;
        if (message.length) {
            NSRect bounds = [message boundingRectWithSize:NSMakeSize(296, CGFLOAT_MAX)
                options:NSStringDrawingUsesLineFragmentOrigin
                attributes:@{NSFontAttributeName: [NSFont systemFontOfSize:11]}];
            messageHeight = ceil(bounds.size.height) + 8;
        }
        CGFloat height = 44 + messageHeight + windows.count * 76;
        RLBCard *card = [[RLBCard alloc] initWithFrame:NSMakeRect(14, y, 332, height)];
        [card addSubview:RLBLabel(provider[@"name"], NSMakeRect(16, 12, 296, 22),
            [NSFont systemFontOfSize:14 weight:NSFontWeightSemibold], NSColor.labelColor)];
        CGFloat rowY = 40;
        if (message.length) {
            NSTextField *status = RLBLabel(message, NSMakeRect(16, rowY, 296, messageHeight),
                [NSFont systemFontOfSize:11], NSColor.secondaryLabelColor);
            status.maximumNumberOfLines = 0;
            status.lineBreakMode = NSLineBreakByWordWrapping;
            [card addSubview:status];
            rowY += messageHeight;
        }
        for (NSDictionary *window in windows) {
            NSButton *toggle = [NSButton checkboxWithTitle:window[@"label"] target:self action:@selector(performAction:)];
            toggle.frame = NSMakeRect(14, rowY, 190, 22);
            toggle.font = [NSFont systemFontOfSize:12];
            toggle.cell.lineBreakMode = NSLineBreakByTruncatingTail;
            toggle.state = [window[@"selected"] boolValue] ? NSControlStateValueOn : NSControlStateValueOff;
            toggle.identifier = window[@"action"];
            toggle.toolTip = [NSString stringWithFormat:@"%@: %@", window[@"label"], window[@"tooltip"]];
            [toggle setAccessibilityLabel:toggle.toolTip];
            [card addSubview:toggle];
            NSInteger usedPercent = [window[@"usedPercent"] integerValue];
            NSTextField *value = RLBLabel(window[@"usage"], NSMakeRect(206, rowY + 2, 110, 20),
                [NSFont monospacedDigitSystemFontOfSize:12 weight:NSFontWeightSemibold], NSColor.labelColor);
            value.alignment = NSTextAlignmentRight;
            [card addSubview:value];
            RLBUsageBar *bar = [[RLBUsageBar alloc] initWithFrame:NSMakeRect(16, rowY + 28, 300, 6)];
            bar.usedPercent = usedPercent;
            [bar setAccessibilityLabel:[NSString stringWithFormat:@"%@: %@", window[@"label"], window[@"usage"]]];
            [card addSubview:bar];
            [card addSubview:RLBLabel([NSString stringWithFormat:@"%@ · %@", window[@"remaining"], window[@"reset"]], NSMakeRect(16, rowY + 40, 300, 18),
                [NSFont systemFontOfSize:11], NSColor.secondaryLabelColor)];
            rowY += 76;
        }
        [document addSubview:card];
        y += height + 12;
    }
    document.frame = NSMakeRect(0, 0, 360, y);
    self.scroll.documentView = document;
    CGFloat availableHeight = (self.popover.contentViewController.view.window.screen ?: NSScreen.mainScreen).visibleFrame.size.height - 100;
    CGFloat height = MIN(y + 52, MIN(640, MAX(200, availableHeight)));
    self.popover.contentSize = NSMakeSize(360, height);
    self.scroll.frame = NSMakeRect(0, 52, 360, height - 52);
    [self.scroll.contentView scrollToPoint:NSMakePoint(0, MIN(scrollPosition.y, MAX(0, y - (height - 52))))];
    [self.scroll reflectScrolledClipView:self.scroll.contentView];
    self.footer.frame = NSMakeRect(0, 0, 360, 52);
    for (NSView *view in self.footer.subviews.copy) [view removeFromSuperview];
    NSBox *separator = [[NSBox alloc] initWithFrame:NSMakeRect(14, 51, 332, 1)];
    separator.boxType = NSBoxSeparator;
    [self.footer addSubview:separator];
    NSButton *settings = [NSButton buttonWithTitle:self.model[@"settings"] target:self action:@selector(showSettings:)];
    settings.bezelStyle = NSBezelStyleRounded;
    settings.keyEquivalent = @",";
    settings.keyEquivalentModifierMask = NSEventModifierFlagCommand;
    settings.frame = NSMakeRect(12, 12, 100, 28);
    [self.footer addSubview:settings];
    [self.footer addSubview:[self button:self.model[@"refresh"] frame:NSMakeRect(130, 12, 100, 28)]];
    [self.footer addSubview:[self button:self.model[@"quit"] frame:NSMakeRect(248, 12, 100, 28)]];
}

- (void)performAction:(id)sender {
    NSString *action = [sender identifier];
    if ([action isEqualToString:self.model[@"quit"][@"action"]]) [self.popover close];
    rlbPanelEvent((char *)action.UTF8String);
}

- (void)showSettings:(NSButton *)sender {
    [self.popover close];
    [self.settings showWindow:sender];
}

@end

static RLBPanelController *panel;

void rlb_panel_update(const char *json) {
    NSData *data = [[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding];
    NSDictionary *model = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (!panel) panel = [RLBPanelController new];
        panel.model = model;
        [panel render];
    });
}


void rlb_panel_toggle(void) {
    NSView *anchor = RLBStatusButton();
    if (panel.popover.shown) {
        [panel.popover close];
    } else if (anchor && panel.model) {
        [panel.popover showRelativeToRect:anchor.bounds ofView:anchor preferredEdge:NSRectEdgeMinY];
        rlbPanelEvent("open");
    }
}

void rlb_panel_close(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [panel.popover close];
        [panel.settings close];
    });
}

void rlb_settings_show(void) {
    dispatch_async(dispatch_get_main_queue(), ^{ [panel showSettings:nil]; });
}
