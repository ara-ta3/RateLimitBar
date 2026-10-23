#import "status_darwin.h"

static NSStatusBarButton *RLBFindStatusButton(NSView *view) {
    if ([view isKindOfClass:NSStatusBarButton.class]) return (NSStatusBarButton *)view;
    for (NSView *child in view.subviews) {
        NSStatusBarButton *button = RLBFindStatusButton(child);
        if (button) return button;
    }
    return nil;
}

NSStatusBarButton *RLBStatusButton(void) {
    for (NSWindow *window in NSApp.windows) {
        NSStatusBarButton *button = RLBFindStatusButton(window.contentView);
        if (button) return button;
    }
    return nil;
}

static NSImage *RLBStatusBar(NSDictionary *provider) {
    return [NSImage imageWithSize:NSMakeSize(22, 7) flipped:NO drawingHandler:^BOOL(NSRect bounds) {
        NSColor *muted = [NSColor colorWithWhite:0.5 alpha:0.5];
        if ([provider[@"unavailable"] boolValue]) {
            [muted setFill];
            NSRectFill(NSMakeRect(7, 3, 8, 1));
            return YES;
        }
        [[NSColor colorWithWhite:0.5 alpha:0.3] setFill];
        [[NSBezierPath bezierPathWithRoundedRect:bounds xRadius:2 yRadius:2] fill];
        NSInteger used = [provider[@"usedPercent"] integerValue];
        NSColor *color = used >= 90 ? NSColor.systemRedColor :
            (used >= 70 ? NSColor.systemYellowColor : NSColor.systemGreenColor);
        if ([provider[@"stale"] boolValue]) color = muted;
        [color setFill];
        NSRect fill = bounds;
        fill.size.width *= MIN(100, MAX(0, used)) / 100.0;
        [[NSBezierPath bezierPathWithRoundedRect:fill xRadius:2 yRadius:2] fill];
        return YES;
    }];
}

NSAttributedString *RLBStatusTitle(NSArray *providers) {
    NSMutableAttributedString *title = [NSMutableAttributedString new];
    NSDictionary *attributes = @{
        NSFontAttributeName: [NSFont systemFontOfSize:10 weight:NSFontWeightSemibold],
        NSForegroundColorAttributeName: NSColor.labelColor
    };
    for (NSDictionary *provider in providers) {
        if (title.length) [title appendAttributedString:[[NSAttributedString alloc] initWithString:@"  " attributes:attributes]];
        [title appendAttributedString:[[NSAttributedString alloc] initWithString:[provider[@"label"] stringByAppendingString:@" "] attributes:attributes]];
        NSTextAttachment *attachment = [NSTextAttachment new];
        attachment.image = RLBStatusBar(provider);
        attachment.bounds = NSMakeRect(0, -1, 22, 7);
        [title appendAttributedString:[NSAttributedString attributedStringWithAttachment:attachment]];
    }
    return title;
}

void rlb_status_update(const char *json) {
    NSData *data = [[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding];
    NSDictionary *model = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
    dispatch_async(dispatch_get_main_queue(), ^{
        NSStatusBarButton *button = RLBStatusButton();
        NSArray *providers = model[@"providers"];
        if ((id)providers == NSNull.null) providers = @[];
        button.title = @"";
        if (providers.count == 0) {
            button.image = [NSImage imageWithSystemSymbolName:@"chart.bar.fill" accessibilityDescription:@"RateLimitBar"];
            button.image.template = YES;
            button.imagePosition = NSImageOnly;
        } else {
            button.image = nil;
            button.imagePosition = NSNoImage;
            button.attributedTitle = RLBStatusTitle(providers);
        }
        button.toolTip = model[@"tooltip"];
        [button setAccessibilityLabel:model[@"tooltip"]];
    });
}
