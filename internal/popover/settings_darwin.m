#import "settings_darwin.h"

extern void rlbPanelEvent(char *action);

@interface RLBSettingsWindow : NSWindow
@end
@implementation RLBSettingsWindow
- (BOOL)performKeyEquivalent:(NSEvent *)event {
    if ((event.modifierFlags & NSEventModifierFlagCommand) && [event.charactersIgnoringModifiers isEqualToString:@"w"]) {
        [self performClose:nil];
        return YES;
    }
    return [super performKeyEquivalent:event];
}
@end

@interface RLBSettingsContent : NSView
@end
@implementation RLBSettingsContent
- (BOOL)isFlipped { return YES; }
@end

static NSTextField *RLBSettingsLabel(NSString *text, NSRect frame, NSFont *font) {
    NSTextField *label = [NSTextField labelWithString:text];
    label.frame = frame;
    label.font = font;
    return label;
}

@interface RLBSettingsController ()
@property NSDictionary *model;
- (void)render;
- (void)performAction:(NSControl *)sender;
- (void)changeLanguage:(NSPopUpButton *)sender;
@end

@implementation RLBSettingsController
- (instancetype)init {
    NSWindow *window = [[RLBSettingsWindow alloc] initWithContentRect:NSMakeRect(0, 0, 560, 400)
        styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable
        backing:NSBackingStoreBuffered defer:NO];
    if ((self = [super initWithWindow:window])) {
        window.releasedWhenClosed = NO;
        window.collectionBehavior = NSWindowCollectionBehaviorMoveToActiveSpace;
        [window center];
    }
    return self;
}

- (void)updateModel:(NSDictionary *)model {
    NSArray *commands = model[@"commands"];
    if (!commands || (id)commands == NSNull.null) commands = @[];
    NSDictionary *settings = @{
        @"commands": commands,
        @"title": model[@"settingsTitle"],
        @"general": model[@"generalTitle"],
        @"sources": model[@"sourcesTitle"],
        @"language": model[@"languageLabel"],
        @"choose": model[@"changeSource"]
    };
    if ([self.model isEqualToDictionary:settings]) return;
    self.model = settings;
    [self render];
}

- (void)showWindow:(id)sender {
    if (self.window.miniaturized) [self.window deminiaturize:nil];
    [self.window makeKeyAndOrderFront:sender];
    [NSApp activateIgnoringOtherApps:YES];
}

- (void)render {
    self.window.title = self.model[@"title"];
    RLBSettingsContent *content = [[RLBSettingsContent alloc] initWithFrame:NSMakeRect(0, 0, 560, 400)];
    NSFont *headingFont = [NSFont systemFontOfSize:13 weight:NSFontWeightSemibold];
    NSFont *labelFont = [NSFont systemFontOfSize:13];
    CGFloat y = 24;
    [content addSubview:RLBSettingsLabel(self.model[@"general"], NSMakeRect(24, y, 512, 22), headingFont)];
    y += 36;
    [content addSubview:RLBSettingsLabel(self.model[@"language"], NSMakeRect(24, y + 4, 140, 22), labelFont)];
    NSPopUpButton *language = [[NSPopUpButton alloc] initWithFrame:NSMakeRect(176, y, 220, 28) pullsDown:NO];
    language.target = self;
    language.action = @selector(changeLanguage:);
    [language setAccessibilityLabel:self.model[@"language"]];
    for (NSDictionary *command in self.model[@"commands"]) {
        if (![command[@"kind"] isEqualToString:@"language"]) continue;
        [language addItemWithTitle:command[@"title"]];
        NSMenuItem *item = language.lastItem;
        item.identifier = command[@"action"];
        if ([command[@"checked"] boolValue]) [language selectItem:item];
    }
    [content addSubview:language];
    y += 44;
    for (NSDictionary *command in self.model[@"commands"]) {
        if (![command[@"kind"] isEqualToString:@"toggle"]) continue;
        NSButton *toggle = [NSButton checkboxWithTitle:command[@"title"] target:self action:@selector(performAction:)];
        toggle.frame = NSMakeRect(24, y, 512, 24);
        toggle.identifier = command[@"action"];
        toggle.state = [command[@"checked"] boolValue] ? NSControlStateValueOn : NSControlStateValueOff;
        [content addSubview:toggle];
        y += 32;
    }
    y += 16;
    NSBox *separator = [[NSBox alloc] initWithFrame:NSMakeRect(24, y, 512, 1)];
    separator.boxType = NSBoxSeparator;
    [content addSubview:separator];
    y += 20;
    [content addSubview:RLBSettingsLabel(self.model[@"sources"], NSMakeRect(24, y, 512, 22), headingFont)];
    y += 36;
    for (NSDictionary *command in self.model[@"commands"]) {
        if (![command[@"kind"] isEqualToString:@"source"]) continue;
        NSString *label = command[@"label"];
        NSButton *button;
        if (label.length) {
            [content addSubview:RLBSettingsLabel(label, NSMakeRect(24, y, 396, 22), headingFont)];
            NSTextField *path = RLBSettingsLabel(command[@"detail"], NSMakeRect(24, y + 24, 396, 22), [NSFont systemFontOfSize:12]);
            path.lineBreakMode = NSLineBreakByTruncatingMiddle;
            path.textColor = NSColor.secondaryLabelColor;
            path.selectable = YES;
            path.toolTip = command[@"detail"];
            [path setAccessibilityLabel:label];
            [content addSubview:path];
            button = [NSButton buttonWithTitle:self.model[@"choose"] target:self action:@selector(performAction:)];
            button.frame = NSMakeRect(436, y + 6, 100, 28);
            [button setAccessibilityLabel:[NSString stringWithFormat:@"%@: %@", label, self.model[@"choose"]]];
            y += 72;
        } else {
            button = [NSButton buttonWithTitle:command[@"title"] target:self action:@selector(performAction:)];
            button.frame = NSMakeRect(24, y + 4, 512, 28);
            y += 44;
        }
        button.bezelStyle = NSBezelStyleRounded;
        button.identifier = command[@"action"];
        [content addSubview:button];
    }
    CGFloat height = y + 16;
    content.frame = NSMakeRect(0, 0, 560, height);
    self.window.contentView = content;
    [self.window setContentSize:NSMakeSize(560, height)];
}

- (void)performAction:(NSControl *)sender {
    rlbPanelEvent((char *)sender.identifier.UTF8String);
}

- (void)changeLanguage:(NSPopUpButton *)sender {
    NSString *action = sender.selectedItem.identifier;
    for (NSDictionary *command in self.model[@"commands"]) {
        if ([command[@"kind"] isEqualToString:@"language"] && [command[@"checked"] boolValue]) {
            [sender selectItemWithTitle:command[@"title"]];
            break;
        }
    }
    rlbPanelEvent((char *)action.UTF8String);
}
@end
